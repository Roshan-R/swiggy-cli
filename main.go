package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/kirsle/configdir"
)

type Orders struct {
	Data struct {
		Orders []struct {
			SharedOrder bool   `json:"sharedOrder"`
			OrderID     int64  `json:"order_id"`
			CustomerId  string `json:"customer_id"`
		} `json:"orders"`
	} `json:"data"`
}

type OrderTracker struct {
	Data struct {
		Configuration struct {
			PollingIntervalSeconds int `json:"polling_interval_seconds"`
		} `json:"configuration"`
		OrderStatusDetails struct {
			StatusMessage       string `json:"status_message"`
			StatusMessageColour string `json:"status_message_colour"`
			BodyLayout          string `json:"body_layout"`
			Messages            []struct {
				Body string `json:"body"`
			} `json:"messages"`
			EtaText    string `json:"eta_text"`
			EtaSubtext string `json:"eta_subtext"`
		} `json:"order_status_details"`
		TrackCrouton struct {
			Title              string `json:"title"`
			ProgressPercentage int    `json:"progress_percentage"`
		} `json:"track_crouton"`
	} `json:"data"`
}

var (
	cookieFile   string
	ErrCookieBad = errors.New("invalid or expired cookie")
)

func WriteCookie(cookieString string) error {
	return os.WriteFile(cookieFile, []byte(cookieString), 0666)
}

func ReadCookie() (string, error) {
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		RefreshCookie(true)
	}
	data, error := os.ReadFile(cookieFile)
	if error != nil {
		panic("Cannot get saved cookie from path: " + cookieFile)
	}

	return strings.TrimSpace(string(data)), error
}

func RefreshCookie(new bool) string {
	if new {
		fmt.Println("Go to the browser and fetch the swiggy cookies here: ")
	} else {
		fmt.Println("The given cookie has expired. Paste a new one here: ")
	}
	reader := bufio.NewReader(os.Stdin)
	cookie, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	err = WriteCookie(cookie)
	if err != nil {
		fmt.Println("Can't save cookie to the path :" + cookieFile)
	}
	return cookie
}

func req(url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	cookie, _ := ReadCookie()

	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:139.0) Gecko/20100101 Firefox/139.0")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cookie", cookie)

	return http.DefaultClient.Do(req)
}

func GetReq(url string) (*http.Response, error) {
	res, err := req(url)
	if err != nil {
		panic(err)
	}
	if len(res.Header["Set-Cookie"]) != 3 {
		RefreshCookie(false)
		return req(url)
	}
	return res, err
}

func GetOrders() (Orders, error) {
	url := "https://www.swiggy.com/dapi/order/all?order_id="
	res, err := GetReq(url)

	if err == errors.New("Cookie expired") {
		res, err = GetReq(url)
	} else if err != nil {
		panic("Could not talk to swiggy api")
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var orders Orders
	if err := json.Unmarshal(body, &orders); err != nil { // Parse []byte to go struct pointer
		panic("Can not unmarshal JSON")
	}
	return orders, nil

}

func TrackOrder(orderId int64, customerId string) (OrderTracker, error) {
	url := "https://www.swiggy.com/dapi/order/trackV4?order_id=" + strconv.FormatInt(orderId, 10) + "&type=full&version=V2&customer_id=" + customerId

	res, _ := GetReq(url)
	body, _ := io.ReadAll(res.Body)

	var orderTracker OrderTracker
	if err := json.Unmarshal(body, &orderTracker); err != nil {
		panic("Can not unmarshal JSON")
	}

	return orderTracker, nil
}

func Init() {

	// Handler to use, to show the cursor
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Printf("\x1b[?25h")
			os.Exit(0)
		}
	}()

	// hide the cursor
	fmt.Printf("\x1b[?25l")
	configPath := configdir.LocalConfig("swiggy-cli")
	err := configdir.MakePath(configPath) // Ensure it exists.
	if err != nil {
		panic(err)
	}
	cookieFile = filepath.Join(configPath, "cookie")
}

type OrderStatus int

const (
	OrderOngoing = iota
	OrderDelivered
)

/*
	Progress bar example
	[Preparing your order][====================>..............................][10mins]
*/

func DisplayStatus(tracker OrderTracker, spinnerString string) OrderStatus {
	title := tracker.Data.TrackCrouton.Title
	if title == "Order Delivered" {
		fmt.Println("Cannot find any active orders to track")
		os.Exit(0)
	}

	//if tracker.Data.TrackCrouton.Title == "Out for delivery" {
	width, _, err := term.GetSize(0)
	if err != nil {
		panic(err)
	}

	// 15
	time := tracker.Data.OrderStatusDetails.EtaText
	// mins
	unit := tracker.Data.OrderStatusDetails.EtaSubtext
	// 15mins
	eta := time + unit

	fmt.Printf("%s [%s]", spinnerString, title)

	progressWidth := width - len(title) - len(eta) - 8
	percentInWidthFloat := float32(progressWidth) / float32(100) * float32(tracker.Data.TrackCrouton.ProgressPercentage)
	percentInWidth := int(percentInWidthFloat)

	fmt.Printf("[")

	// TODO: Add colors to the progress bar
	for x := range progressWidth {
		var char string

		if x < percentInWidth {
			char = "="
		} else if x == percentInWidth {
			char = ">"
		} else {
			char = "."
		}
		fmt.Print(char)
	}
	fmt.Printf("]")
	fmt.Printf("[%s]\r", eta)

	//fmt.Println("width, progresswidth, percentinwidth, percent", width, progressWidth, percentInWidth, tracker.Data.TrackCrouton.ProgressPercentage)
	return OrderOngoing
}

func updateTracker(tracker *OrderTracker, latestOrder int64, customerId string) {
	for {
		trackerEx, _ := TrackOrder(latestOrder, customerId)
		*tracker = trackerEx
		time.Sleep(time.Second * 2)
	}
}

func main() {
	Init()

	orders, _ := GetOrders()
	latestOrder := orders.Data.Orders[0].OrderID
	customerId := orders.Data.Orders[0].CustomerId

	var tracker OrderTracker

	go updateTracker(&tracker, latestOrder, customerId)

	index := 0
	for {
		progressStates := []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}
		DisplayStatus(tracker, progressStates[index])

		index++
		if index == len(progressStates)-2 {
			index = 0
		}
		time.Sleep(time.Millisecond * 300)
	}

}
