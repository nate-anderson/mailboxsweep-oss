package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"strconv"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "mailboxsweep",
}

var (
	username string
	host     string
	password string
	port     int

	folderName string
	path       []string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "IMAP username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "IMAP password")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "IMAP host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", 993, "IMAP port")

	listMessagesCmd.PersistentFlags().StringVar(&folderName, "folder", "", "the inbox folder to read from")
	classifyMessagesCmd.PersistentFlags().StringVar(&folderName, "folder", "", "the inbox folder to read from")
	deleteCmd.PersistentFlags().StringVar(&folderName, "folder", "", "the inbox folder to delete from")
	queryCmd.PersistentFlags().StringSliceVar(&path, "path", []string{}, "the path to the node to query")
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

		provider, err := inbox.NewIMAPProvider(host, port, nil)
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

		provider, err := inbox.NewIMAPProvider(host, port, nil)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Connected!")

		mailbox, err := provider.Login(username, password)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Logged in!")

		folder, err := mailbox.Folder(folderName)
		if err != nil {
			log.Fatal(err)
		}

		count := folder.Count()
		log.Printf("Messages: %d\n", count)

		msgChan := make(chan inbox.Message)
		errChan := make(chan error)
		go folder.Messages(1, count, msgChan, errChan)

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

var classifyMessagesCmd = &cobra.Command{
	Use:   "classify",
	Short: "Classify inbox messages",
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

		provider, err := inbox.NewIMAPProvider(host, port, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		mailbox, err := provider.Login(username, password)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Logged in!")

		folder, err := mailbox.Folder(folderName)
		if err != nil {
			log.Fatal(err)
		}

		count := folder.Count()
		log.Printf("Messages: %d\n", count)

		repo := classification.NewFileClassificationRepository("testdata")
		organizer := classification.NewOrganizer(mailbox, repo, []string{folderName}, []classification.Classifier{
			classification.NewSenderDomainClassifier(),
			classification.NewSenderUsernameClassifier(),
		})

		err = organizer.Organize(context.Background())
		if err != nil {
			log.Fatalf("Organize failed: %s", err)
		}

		log.Println("Done!")
	},
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "query a stored inbox summary",
	Run: func(cmd *cobra.Command, args []string) {
		if username == "" {
			log.Fatal("username is required")
		}

		repo := classification.NewFileClassificationRepository("testdata")
		classification, err := repo.FetchSummary(context.Background(), username)
		if err != nil {
			log.Fatal(err)
		}

		data, err := json.MarshalIndent(classification, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(data))
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete some UIDs from the inbox",
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

		provider, err := inbox.NewIMAPProvider(host, port, nil)
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

		uids := []string{}
		for _, arg := range args {
			uid, err := strconv.Atoi(arg)
			if err != nil {
				log.Fatal(err)
			}
			uids = append(uids, strconv.Itoa(uid))
		}

		if err := folder.Delete(uids); err != nil {
			log.Fatal(err)
		}

		log.Println("Done!")
	},
}

func main() {
	rootCmd.AddCommand(listMailboxesCmd)
	rootCmd.AddCommand(listMessagesCmd)
	rootCmd.AddCommand(classifyMessagesCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deleteCmd)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
