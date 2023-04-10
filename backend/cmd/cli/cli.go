package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/login"
	"mailboxsweep/metering"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

		ctx := context.Background()

		provider, err := inbox.NewIMAPProvider(ctx, host, port, username, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(ctx); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		cred := inboxcredential.NewIMAPInboxCredential(username, password)

		inbox, err := provider.Login(ctx, username, cred)
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

		ctx := context.Background()

		awsCfg := mailboxsweep.AWSServiceConfig{
			ServiceHost:   aws.String("localhost"),
			ServicePort:   4566,
			ServiceRegion: "us-west-2",
		}

		cl, err := mailboxsweep.ConnectAWSService(
			awsCfg,
			dynamodb.NewFromConfig,
		)
		if err != nil {
			log.Fatal(err)
		}

		usageRepo := metering.NewUsageRepository(cl, "mailboxsweep")

		quotaBytes := metering.Megabyte * 1800

		provider, err := inbox.NewIMAPProvider(ctx, host, port, username, usageRepo, &quotaBytes)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Connected!")

		cred := inboxcredential.NewIMAPInboxCredential(username, password)

		mailbox, err := provider.Login(ctx, username, cred)
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
		go folder.Messages(ctx, 1, count, msgChan, errChan)

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

		ctx := context.Background()

		provider, err := inbox.NewIMAPProvider(ctx, host, port, username, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(ctx); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		cred := inboxcredential.NewIMAPInboxCredential(username, password)

		mailbox, err := provider.Login(ctx, username, cred)
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

		repo := classification.NewFileClassificationRepository("testdata", 50)
		organizer := classification.NewOrganizer(mailbox, repo, []string{folderName}, []classification.Classifier{
			classification.NewSenderDomainClassifier(),
			classification.NewSenderUsernameClassifier(),
		})

		_, err = organizer.Organize(context.Background())
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

		repo := classification.NewFileClassificationRepository("testdata", 50)
		classification, err := repo.FetchSummary(context.Background(), username, 0)
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

		ctx := context.Background()

		provider, err := inbox.NewIMAPProvider(ctx, host, port, username, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := provider.Connect(ctx); err != nil {
			log.Fatal(err)
		}
		log.Println("Connected!")

		cred := inboxcredential.NewIMAPInboxCredential(username, password)

		inbox, err := provider.Login(ctx, username, cred)
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

type cmdConfig struct {
	GoogleClientID        string `mapstructure:"GOOGLE_OAUTH_CLIENT_ID"`
	GoogleClientSecret    string `mapstructure:"GOOGLE_OAUTH_CLIENT_SECRET"`
	GoogleAuthRedirectURL string `mapstructure:"GOOGLE_OAUTH_REDIRECT_URL"`

	DynamoDBHost   *string `mapstructure:"DYNAMODB_HOST"`
	DynamoDBPort   int     `mapstructure:"DYNAMODB_PORT"`
	DynamoDBRegion string  `mapstructure:"DYNAMODB_REGION"`
	DynamoDBTable  string  `mapstructure:"DYNAMODB_TABLE"`

	CredentialCipherKeyBase64 string `mapstructure:"CREDENTIAL_CIPHER_KEY_BASE64"`
}

var cfg cmdConfig

var gmailMessageSummary = &cobra.Command{
	Use:   "gmail-summary",
	Short: "Summarize gmail messages",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetConfigType("env")
		viper.SetConfigFile(".env")
		viper.AutomaticEnv()
		err := viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}

		if err := viper.Unmarshal(&cfg); err != nil {
			log.Fatal(err)
		}

		ctx := context.Background()
		provider := inbox.NewGmailProvider(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret)

		db, err := mailboxsweep.ConnectAWSService(
			mailboxsweep.AWSServiceConfig{
				ServiceHost:   cfg.DynamoDBHost,
				ServicePort:   cfg.DynamoDBPort,
				ServiceRegion: cfg.DynamoDBRegion,
			},
			dynamodb.NewFromConfig,
		)
		if err != nil {
			log.Fatal(err)
		}

		// let user log in
		tokenRepo := login.NewDynamoGmailTokenRepository(db, cfg.DynamoDBTable)

		credentialRepo := inboxcredential.NewDynamoInboxCredentialRepository(db, cfg.DynamoDBTable)

		cipherKey, err := base64.RawStdEncoding.DecodeString(cfg.CredentialCipherKeyBase64)
		if err != nil {
			log.Fatalf("failed to decode configured AES Cipher key: %s", err)
		}
		credentialService, err := inboxcredential.NewService(credentialRepo, cipherKey)
		if err != nil {
			log.Fatal(err)
		}

		redirectURL := "https://localhost:3001/login-gmail"
		gmailLoginService := login.NewGmailLoginService(tokenRepo, credentialService, cfg.GoogleClientID, cfg.GoogleClientSecret, redirectURL)

		config := oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  redirectURL,
			Scopes:       mailboxsweep.GoogleAPIScopes,
		}

		authedCtx := mailboxsweep.AuthorizeJobContext(ctx, username)
		gmailCredential, err := credentialService.FindCredential(authedCtx)
		if err == nil {
			gmailCredential, err = credentialService.DecryptCredential(authedCtx, gmailCredential)
			if err != nil {
				log.Fatal(err)
			}
		} else if errors.Is(err, mailboxsweep.ErrNotFound) {

			url, _, err := gmailLoginService.GetAuthURL(ctx, username)
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("Go to URL %s to authenticate\n", url)

			log.Println("Enter auth code: ")
			var authCode string
			if _, err := fmt.Scan(&authCode); err != nil {
				log.Fatalf("Unable to read authorization code: %v", err)
			}

			fmt.Printf("Got auth code '%s'\n", authCode)
			tok, err := config.Exchange(context.TODO(), authCode)
			if err != nil {
				log.Fatalf("Unable to retrieve token from web: %v", err)
			}

			gmailCredential = inboxcredential.NewGmailInboxCredential(username, tok.AccessToken, tok.RefreshToken, tok.Expiry)
			err = credentialService.StoreCredential(authedCtx, gmailCredential)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}

		myInbox, err := provider.Login(ctx, username, gmailCredential)
		if err != nil {
			log.Fatal(err)
		}

		folder, err := myInbox.Folder("INBOX")
		if err != nil {
			log.Fatal(err)
		}

		count := folder.Count()
		mChan := make(chan inbox.Message)
		eChan := make(chan error)
		go folder.Messages(ctx, 0, uint32(count), mChan, eChan)

		readCount := 0

		for {
			select {
			case e := <-eChan:
				if e == nil {
					fmt.Printf("Finished read!")
					return
				}
				fmt.Printf("Error from Messages read: %s\n", e)
				return
			case m := <-mChan:
				fmt.Printf("Got message %s (from %s)\n", m.Subject, m.Sender.Address)
				readCount++
			}
		}
	},
}

func main() {
	rootCmd.AddCommand(listMailboxesCmd)
	rootCmd.AddCommand(listMessagesCmd)
	rootCmd.AddCommand(classifyMessagesCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(gmailMessageSummary)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
