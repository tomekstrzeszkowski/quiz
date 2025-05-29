# Quiz
P2P quiz app with mdns peer discovery 

# Delve Debugger

## Install

 - Run `go env | rg GOBIN`, If the above command prints, `GOBIN=""`, it means that `GOBIN` is not set. Run export `GOBIN=~/go/bin/` command to set GOBIN.
 - Add `GOBIN` to the `PATH` by running export `PATH=$PATH:~/go/bin`
 - Install dlv `go install github.com/go-delve/delve/cmd/dlv@latest`, verify `dlv version`

## Debugging

 - In the first terminal run `dlv debug --headless --listen=:2345 --api-version=2 -- -l 10000`
 - In the secound terminal `dlv connect :2345`
 - Go back to the first terminal, user input and breakpoints should work

Example
Terminal 2: 
 - `break main.go:31`
 - `print *Nick`
 - `continue`

Terminal 1:
 - Program is running, type `all`