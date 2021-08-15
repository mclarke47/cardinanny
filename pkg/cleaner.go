package pkg

import "fmt"

type PromCleaner struct {
}

func query(labelName string) string {
	return fmt.Sprintf("{%s=~\".+\"}", labelName)
}

func (p *PromCleaner) Clean(labelsToDrop string) error {

	// for _, labels := range jobToLabelToDrop {
	// 	for _, v := range labels {
	// 		err = v1api.DeleteSeries(context.Background(), []string{query(v)}, time.Now().Add(-time.Hour), time.Now())

	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		if resp.StatusCode != 200 {
	// 			defer resp.Body.Close()
	// 			b, err := ioutil.ReadAll(resp.Body)
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			log.Fatalf("Expected status code 200 but was %d, body: %s", resp.StatusCode, b)
	// 		}
	// 	}

	// }

	// err = v1api.CleanTombstones(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	return nil
}
