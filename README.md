# Onlinesim GO API

Wrapper for automatic reception of SMS-messages by onlinesim.ru

## Installation

Require this package in your `package.json` or install it by running:
```bash
go get github.com/s00d/onlinesim-go-api
```

### Example
```go
package main

import (
    "github.com/s00d/onlinesim-go-api"
)

func main() {
    client := NewClient("", "en", "").numbers()
    
    error, data := client.get("vkcom", 7)
    if error != nil {
        panic(error)
    }

    println("end")
    println(fmt.Sprintf("%+v\n", data))
}
```

## Documentation

All documentation is in the wiki of this project - **[Documentation](https://github.com/s00d/onlinesim-go-api/wiki)**

## Bugs

If you have any problems, please create Issues [here](https://github.com/s00d/onlinesim-go-api/issues)   
