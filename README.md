# swiggy-cli

![image](https://github.com/user-attachments/assets/a5de87eb-bbf3-45bf-b4c2-6a59f2a5a28c)

swiggy-cli is a cli application to track your swiggy order. It makes use of user provided browser cookies and calls the SwiggyAPIs to fetch and show data.
It checks the status of the last order you placed, and if it's an ongoing order a simple downloader-like progress bar shows up with real time updates.

The browser cookie is stored as a plain text file to ~/.config/swiggy-cli/cookie

## Installation

### Pre-Built Binaries

Pre-built binaries are available for most operating systems and architectures are available in the latest release. 
To use them, simply download the binary and run. To make the program available from every directory, make sure to move
the binary to a location that is in the PATH.

### From Source

you could clone the repo and build the binary yourself

```bash
git clone https://github.com/Roshan-R/swiggy-cli
cd swiggy-cli
go build .
```

this will produce a binary in the current directory.

## Contributing

Contributing to the project is more than welcome, if by code or ideas :)
