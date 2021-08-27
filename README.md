# sockd

##  Alternative
This is still incomplete implementation.
Better [alternative](https://github.com/joewalnes/websocketd).

## Usage
Get help message
```bash
go run main.go -help
```
## Example
### Run server with default parameters
```bash
go run main.go
```
Will start server on `localhost:8080` and will stream the output of `ls` command.

### Or provide parameters for the server
```bash
go run main.go -address="0.0.0.0" -port=8000 -script=pwd
```
Will start server on `0.0.0.0:8000` and will stream the output of `pwd` command.
