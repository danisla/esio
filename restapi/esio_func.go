package restapi

import (
	"encoding/json"
	"log"
	"fmt"
	"net/http"
	"path"
	"sort"
	"time"

	errors "github.com/go-openapi/errors"
	strftime "github.com/hhkbp2/go-strftime"
)

// Create a list of indices to be restored from the given start,end range.
// Snapshots are derived from the given repoPattern and discritized at intervals of given indexResolution
func makeIndexListFromRange(start time.Time, end time.Time, indexResolution string, repoPattern string) ([]string, error) {
	a := make([]string, 0)

	// starting from start time, make index pattern
	// Increment start time by IndexResolution
	// Add next interval to list until time exceedes end time.

	var t = start

	for t.Before(end) {
		a = append(a, strftime.Format(repoPattern, t))
		switch indexResolution {
			case "day": t = t.AddDate(0,0,1)
			case "month": t = t.AddDate(0,1,0)
			case "year": t = t.AddDate(1,0,0)
			default:
				return a, errors.New(400, "Invalid index resolution: " + indexResolution)
		}
	}
	return a, nil
}

// Verifies each index pattern in given list is found on the ES cluster.
func validateSnapshotIndex(repoPattern string) (bool, error) {
	repo := path.Dir(repoPattern)
	target := path.Base(repoPattern)
	url := fmt.Sprintf("%s/_snapshot/%s", myFlags.EsHost, repo)

	log.Println("Checking snapshot: " + url + " for index: " + target)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return false, errors.New(500, fmt.Sprintf("Error building http request: %s", err))
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		//
		return false, errors.New(500, fmt.Sprintf("Error making client request: %s", err))
	}

	defer resp.Body.Close()

	var snap SnapshotResponse

	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		// TODO: need to test this
		return false, errors.New(500, fmt.Sprintf("Error decoding ES JSON response for url: %s", url))
	}

	if len(snap.Snapshots) == 0 {
		return false, errors.New(404, fmt.Sprintf("No snapshots found in repo: %s", repo))
	}

	for _, snapshot := range snap.Snapshots {
		var indices = snapshot.Indices
		sort.Strings(indices)

		i := sort.Search(len(indices),
		    func(i int) bool { return indices[i] >= target })
		if i < len(indices) && indices[i] == target {
			if snapshot.State != "SUCCESS" {
				return false, errors.New(400, fmt.Sprintf("Snapshot state was not 'SUCCESS': %s", snap))
			}
			return true, nil
		}
	}

	return false, errors.New(404, fmt.Sprintf("Index with name '%s' not found in repo: '%s'", target, snap))
}
