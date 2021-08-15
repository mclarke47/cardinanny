package main

import (
	"context"
	"fmt"
	"github.com/mclarke47/cardinanny/pkg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func query(labelName string) string {
	return fmt.Sprintf("{%s=~\".+\"}", labelName)
}

func main() {

	httpClient := http.Client{}

	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})

	if err != nil {
		log.Fatal(err)
	}

	v1api := v1.NewAPI(client)

	cs := pkg.CardinalityScanner{
		PromAPI: v1api,
	}

	jobToLabelToDrop, err := cs.Scan(context.TODO(), 100)
	if err != nil {
		log.Fatal(err)
	}

	if len(jobToLabelToDrop) == 0 {
		log.Println("No config changed required")
		os.Exit(0)
	}

	pcr := pkg.PromConfigRewriter{
		PromAPI: v1api,
	}

	configPath := "./prometheus.yml"

	err = pcr.DropLabelsInJobs(context.TODO(), jobToLabelToDrop, configPath)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := httpClient.Post("http://localhost:9090/-/reload", "", nil)

	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Expected status code 200 but was %d, body: %s", resp.StatusCode, b)
	}

	for _, labels := range jobToLabelToDrop {
		for _, v := range labels {
			err = v1api.DeleteSeries(context.Background(), []string{query(v)}, time.Now().Add(-time.Hour), time.Now())

			if err != nil {
				log.Fatal(err)
			}
			if resp.StatusCode != 200 {
				defer resp.Body.Close()
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				log.Fatalf("Expected status code 200 but was %d, body: %s", resp.StatusCode, b)
			}
		}

	}

	err = v1api.CleanTombstones(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// println(newConfig.String())

	// flags, err := v1api.Flags(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%v\n",flags)

}
