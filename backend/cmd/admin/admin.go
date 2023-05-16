package main

import (
	"fmt"
	"log"
	"mailboxsweep"
	"mailboxsweep/admin"
	"mailboxsweep/changeset"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nate-anderson/fig/v2"
)

type dashConfig struct {
	AdminHTTPPort  int     `fig:"ADMIN_HTTP_PORT" required:"true"`
	Env            string  `fig:"ENV" default:"prod"`
	DynamoDBHost   *string `fig:"DYNAMODB_HOST"`
	DynamoDBPort   int     `fig:"DYNAMODB_PORT"`
	DynamoDBRegion string  `fig:"DYNAMODB_REGION" default:"us-west-2"`

	DynamoDBTable          string `fig:"DYNAMODB_TABLE" required:"true"`
	ChangeSetPageIDHashMod int64  `fig:"CHANGESET_PAGE_ID_HASH_MOD" default:"11"`
}

func main() {

	var cfg dashConfig
	envDriver, err := fig.NewOptionalFileEnvironmentDriver(".env")
	if err != nil {
		log.Fatal(err)
	}

	conf := fig.New(envDriver)
	if err := conf.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

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

	changesetRepo, err := changeset.NewDynamoChangesetRepository(db, cfg.DynamoDBTable, uint64(cfg.ChangeSetPageIDHashMod))
	if err != nil {
		log.Fatal(err)
	}

	changesetService := changeset.NewDefaultService(changesetRepo)

	dash := admin.Dashboard(
		cfg.Env,
		changesetService,
	)

	addr := fmt.Sprintf(":%d", cfg.AdminHTTPPort)
	log.Printf("Serving admin dashboard at %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, dash))
}
