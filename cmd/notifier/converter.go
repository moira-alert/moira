package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/moira-alert/moira-alert"
)

func convertDatabase(db moira.Database) {
	fmt.Println("This will convert all telegram contacts from @ notation to #.")
	fmt.Print("Continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	if !strings.HasPrefix(text, "Y") && !strings.HasPrefix(text, "y") {
		fmt.Println("Aborted")
		os.Exit(0)
	}

	res, _ := db.GetAllContacts()
	for _, contact := range res {
		if contact.Type == "telegram" && strings.HasPrefix(contact.Value, "@") {
			contact.Value = fmt.Sprintf("#%v", contact.Value[1:])
			db.WriteContact(contact)
		}
	}
	os.Exit(0)
}
