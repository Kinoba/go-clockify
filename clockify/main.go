/*

The toggl command will display a user's Clockify account information.

Usage:
    clockify API_TOKEN

The API token can be retrieved from a user's account information page at clockify.me.

*/
package main

import (
	"encoding/json"
	"os"

	"github.com/kinoba/go-clockify"
)

func main() {
	if len(os.Args) != 2 {
		println("usage:", os.Args[0], "API_TOKEN")
		return
	}

	session := clockify.OpenSession(os.Args[1])

	// Get account
	account, err := session.GetAccount()
	if err != nil {
		println("error:", err)
		return
	}
	
	data, err := json.MarshalIndent(&account, "", "    ")
	println("account:", string(data))
}
