package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/unknown/lib"
)

func main() {
	fmt.Println(os.Args, sort.Strings, strings.TrimSpace, resty.New(), lib.Version())
}
