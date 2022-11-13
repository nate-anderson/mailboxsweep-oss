package main

import (
	"log"
	"mailsweeper"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "mailsweeper",
}

var (
	username string
	host     string
	password string
	port     int

	folderName string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "IMAP username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "IMAP password")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "IMAP host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", 993, "IMAP port")

	listMessagesCmd.PersistentFlags().StringVar(&folderName, "folder", "", "the inbox folder to read from")
}

var listMailboxesCmd = &cobra.Command{
	Use:   "mailboxes",
	Short: "List mailboxes associated with this inbox",
	Run: func(cmd *cobra.Command, args []string) {
		if host == "" {
			log.Fatal("host is required")
		}

		if username == "" {
			log.Fatal("username is required")
		}

		if port == 0 {
			log.Fatal("port must be greater than zero")
		}

		if password == "" {
			log.Fatal("password is required")
		}

		provider, err := mailsweeper.NewSMTPProvider(host, port)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		inbox, err := provider.Login(username, password)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Logged in!")

		folders, err := inbox.Folders()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(folders)
	},
}

var listMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List messages",
	Run: func(cmd *cobra.Command, args []string) {
		if host == "" {
			log.Fatal("host is required")
		}

		if username == "" {
			log.Fatal("username is required")
		}

		if port == 0 {
			log.Fatal("port must be greater than zero")
		}

		if password == "" {
			log.Fatal("password is required")
		}

		provider, err := mailsweeper.NewSMTPProvider(host, port)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		inbox, err := provider.Login(username, password)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Logged in!")

		folder, err := inbox.Folder(folderName)
		if err != nil {
			log.Fatal(err)
		}

		count := folder.Count()
		log.Printf("Messages: %d\n", count)

		msgChan := make(chan mailsweeper.Message)
		errChan := make(chan error)
		go folder.Messages(1, 10, msgChan, errChan)

		go func() {
			for m := range msgChan {
				log.Printf("%+v \n", m)
			}
		}()

		if err := <-errChan; err != nil {
			panic(err)
		}
		log.Println("Exited!")

	},
}

func main() {
	rootCmd.AddCommand(listMailboxesCmd)
	rootCmd.AddCommand(listMessagesCmd)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
