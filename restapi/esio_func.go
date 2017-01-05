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

	"github.com/danisla/esio/models"
)

type SnapshotResponse struct {
	Snapshots []Snapshot `json:"snapshots"`
}

type Snapshot struct {
	Snapshot 	string 		`json:"snapshot"`
	VersionId int 	 		`json:"version_id"`
	Indices   []string 	`json:"indices"`
	State     string    `json:"state"`
}

type CatIndex struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Index        string `json:"index"`
	Primaries    string `json:"pri"`
	Replicas     string `json:"rep"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

var restoreQueue *Queue
var deleteQueue *Queue

func initQueues() {
	restoreQueue = NewQueue(1)
	deleteQueue = NewQueue(1)
}

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
		return false, errors.New(500, fmt.Sprintf("Error building http request: %s", err))
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
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

		if stringInList(indices, target) {
			if snapshot.State != "SUCCESS" {
				return false, errors.New(400, fmt.Sprintf("Snapshot state was not 'SUCCESS': %s", snap))
			}
			return true, nil
		}
	}

	return false, errors.New(404, fmt.Sprintf("Index with name '%s' not found in repo: '%s'", target, snap))
}

func getIndices() ([]CatIndex, error) {
	// var cat CatIndices
	cat := make([]CatIndex,0)


	url := fmt.Sprintf("%s/_cat/indices?format=json", myFlags.EsHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cat, errors.New(500, fmt.Sprintf("Error building http request: %s", err))
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return cat, errors.New(500, fmt.Sprintf("Error making client request: %s", err))
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		// TODO: need to test this
		return cat, errors.New(500, fmt.Sprintf("Error decoding ES JSON response for url: %s", url))
	}

	return cat, nil
}

// Takes a list of indices and matches it against the found indices
// Populates the []Ready, []Pending and []Restoring arrays of the IndiceStatus struct.
func makeIndexStatus(indices []string) (models.IndiceStatus, error) {
	var status = &models.IndiceStatus{Pending: make([]string, 0), Ready: make([]string, 0), Restoring: make([]string, 0), Deleting: make([]string, 0)}

	onlineIndices, err := getIndices()
	if err != nil {
		return *status, errors.New(500, fmt.Sprintf("Could not GET _cat/indices from Elasticsearch: %s", err))
	}

	var target = ""
	var found = false

	// Find all indices that are ready (open and green or yellow) or restoring (open and red)
	for _, onlineIndice := range onlineIndices {

		target = onlineIndice.Index

		// Verify indice is in list of requested indices.
		found = stringInList(indices, target)

		if found {
			if onlineIndice.Status != "open" {
				return *status, errors.New(500, fmt.Sprintf("Found existing indice on cluster that was not 'open': %s", target))
			}

			if onlineIndice.Health == "green" || onlineIndice.Health == "yellow" {
				status.Ready = append(status.Ready, target)
			} else if onlineIndice.Health == "red" {
				status.Restoring = append(status.Restoring, target)
			} else {
				return *status, errors.New(500, fmt.Sprintf("Found online index: '%s' with invalid Health state '%s'", target, onlineIndice.Health))
			}
		}
	}

	// Find all indices that are pending (not found in onlineIndices)
	allOnlineIndices := concat(status.Ready, status.Restoring)

	for _, indice := range indices {
		// Verify index is not in the Ready or Restoring lists
		found = false
		target = path.Base(indice)
		found = stringInList(allOnlineIndices, target)
		queued := restoreQueue.Contains(target)
		deleting := deleteQueue.Contains(target)

		if !found && !queued && !deleting {
			status.Pending = append(status.Pending, target)
		}

		if queued {
			status.Restoring = append(status.Restoring, target)
		}

		if deleting {
			status.Deleting = append(status.Deleting, target)
		}
	}

	return *status, nil
}

func deleteIndices(indices []string) (bool, error) {
	// Create the IndexStatus data structure
	indiceStatus, err := makeIndexStatus(indices)
	if err != nil {
		return false, errors.New(500, fmt.Sprintf("Error comparing online indices with snapshots list: %s", err))
	}

	var deleting = false

	for _, indice := range indices {
		target := path.Base(indice)
		queued := stringInList(indiceStatus.Deleting, target)

		if stringInList(indiceStatus.Ready, target) && !queued {
			log.Println(fmt.Sprintf("Deleting online index: %s", target))
			deleteQueue.Push(&Node{target})
			deleting = true
		}
	}

	return deleting, nil
}

func stringInList(l []string, target string) bool {
	i := sort.Search(len(l),
			func(i int) bool { return l[i] >= target })
	if i < len(l) && l[i] == target {
		return true
	}
	return false
}

func concat(old1, old2 []string) []string {
	newslice := make([]string, len(old1) + len(old2))
	copy(newslice, old1)
	copy(newslice[len(old1):], old2)
	return newslice
}

func parseTimeRange(startInput int64, endInput int64) (time.Time, time.Time, error) {
	utc, _ := time.LoadLocation("UTC")
	start := time.Unix(startInput, 0)
	end := time.Unix(endInput, 0)

	// Parse the start time
	if startInput < 0 {
		return start, end, errors.New(400, "Start time must be greater than 0")
	}
	start = start.In(utc)

	// Parse the end time
	if endInput < 0 {
		return start, end, errors.New(400, "End time must be greater than 0")
	}
	end = end.In(utc)

	// Time range must be valid
	if startInput >= endInput {
		return start, end, errors.New(400, "Start time must be less than end time")
	}
	return start, end, nil
}
