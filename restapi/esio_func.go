package restapi

import (
	"encoding/json"
	"log"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	errors "github.com/go-openapi/errors"
	strftime "github.com/hhkbp2/go-strftime"
	elastic "gopkg.in/olivere/elastic.v2"

	"github.com/danisla/esio/models"
)

type EsAcknowledgedResponse struct {
	Acknowledged bool `json:acknowledged`
}

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

type SnapshotRestoreResponse struct {
	Snapshot SnapshotRestore `json:"snapshot"`
}

type SnapshotRestore struct {
	Snapshot string         `json:"snapshot"`
	Indices  []string       `json:"indices"`
	Shards   SnapshotShards `json:"shards"`
}

type SnapshotShards struct {
	Total      int `json:"total"`
	Failed     int `json:"failed"`
	Successful int `json:"successful"`
}

var restoreQueue *Queue
var deleteQueue *Queue

func initQueues() {
	restoreQueue = NewQueue(1)
	deleteQueue = NewQueue(1)

	// Start queue workers

	// Restore queue worker
	go func() {
		for {
			// Restores 1 index at a time based on what is in the restoreQueue
			// TODO dequeue all available, then group by repo/snapshot and perform bulk restore.

			for restoreQueue.count > 0 {
				node := restoreQueue.Pop()
				index := node.Value

				// log.Println(fmt.Sprintf("Restoring index: %s, remaining in queue: %d", index, restoreQueue.count))
				time.Sleep(1000 * time.Millisecond)

				res, err := restoreSnapshot(index)
				if err != nil {
					log.Println(fmt.Sprintf("ERROR: could not restore index: %s, error: %s", index, err))
				} else if !stringInList(res.Indices, path.Base(index)) {
					log.Println(fmt.Sprintf("ERROR: index was not in list of restored indices: %s", res.Indices))
				} else if res.Shards.Successful != res.Shards.Total {
					log.Println(fmt.Sprintf("ERROR: not all shards for index '%s' were successfully recovered.", index))
				} else {
					log.Println(fmt.Sprintf("Successfully recovered index: %s", index))
				}

			}
			time.Sleep(2000 * time.Millisecond)
		}
  }()

	// Delete queue worker
	go func() {
		for {
			for deleteQueue.count > 0 {
				node := deleteQueue.Pop()
				index := node.Value

				time.Sleep(1000 * time.Millisecond)

				client, err := elastic.NewClient(
					elastic.SetURL(myFlags.EsHost),
					elastic.SetHealthcheck(false),
					elastic.SetSniff(false))
				if err != nil {
					log.Println(fmt.Sprintf("ERROR creating Elastic client for index delete: %s", err))
				} else {
					_, err = client.DeleteIndex(path.Base(index)).Do()
					if err != nil {
						log.Println(fmt.Sprintf("ERROR deleting ES index '%s': %s", path.Base(index), err))
					} else {
						log.Println(fmt.Sprintf("Successfully deleted index: %s", index))
					}
				}
			}
			time.Sleep(2000 * time.Millisecond)
		}
	}()
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
	endpoint := fmt.Sprintf("%s/_snapshot/%s", myFlags.EsHost, repo)

	log.Println("Checking snapshot: " + endpoint + " for index: " + target)

	req, err := http.NewRequest("GET", endpoint, nil)
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
		return false, errors.New(500, fmt.Sprintf("Error decoding ES JSON response for url: %s", endpoint))
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

func restoreSnapshot(snap string) (*SnapshotRestore, error) {
	repo := path.Dir(snap)
	indices := path.Base(snap)

	log.Println(fmt.Sprintf("Restoring Snapshot from repo: %s with indices: %s", repo, indices))

	endpoint := fmt.Sprintf("%s/_snapshot/%s/_restore?wait_for_completion=true", myFlags.EsHost, repo)

	data := fmt.Sprintf(`{"indices":"%s"}`,indices)
  buf := strings.NewReader(data)
	resp, err := http.Post(endpoint, "application/json", buf)
	if err != nil {
		return nil, errors.New(500, fmt.Sprintf("HTTP Request error on POST %s", endpoint))
	}

	defer resp.Body.Close()

	var snapRestore SnapshotRestoreResponse

	if err := json.NewDecoder(resp.Body).Decode(&snapRestore); err != nil {
		// TODO: need to test this
		return &snapRestore.Snapshot, errors.New(500, fmt.Sprintf("Error decoding ES JSON response for url: %s", endpoint))
	}

	return &snapRestore.Snapshot, nil
}

func getIndices() ([]CatIndex, error) {
	// var cat CatIndices
	cat := make([]CatIndex,0)


	endpoint := fmt.Sprintf("%s/_cat/indices?format=json", myFlags.EsHost)

	req, err := http.NewRequest("GET", endpoint, nil)
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
		return cat, errors.New(500, fmt.Sprintf("Error decoding ES JSON response for url: %s", endpoint))
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

	var found = false

	// Find all indices that are ready (open and green or yellow) or restoring (open and red)
	for _, onlineIndice := range onlineIndices {

		found = false
		var match = ""

		// Match online index to availalbe indice in snapshot repo.
		for _, i := range indices {
			if path.Base(i) == onlineIndice.Index {
				match = i
				found = true
				break
			}
		}

		if found {
			if onlineIndice.Status != "open" {
				return *status, errors.New(500, fmt.Sprintf("Found existing indice on cluster that was not 'open': %s", match))
			}

			if onlineIndice.Health == "green" || onlineIndice.Health == "yellow" {
				status.Ready = append(status.Ready, match)
			} else if onlineIndice.Health == "red" {
				status.Restoring = append(status.Restoring, match)
			} else {
				return *status, errors.New(500, fmt.Sprintf("Found online index: '%s' with invalid Health state '%s'", match, onlineIndice.Health))
			}
		}
	}

	// Find all indices that are pending (not found in onlineIndices)
	allOnlineIndices := concat(status.Ready, status.Restoring)

	for _, indice := range indices {
		// Verify index is not in the Ready or Restoring lists
		found = false
		found = stringInList(allOnlineIndices, indice)
		queued := restoreQueue.Contains(indice)
		deleting := deleteQueue.Contains(indice)

		if !found && !queued && !deleting {
			status.Pending = append(status.Pending, indice)
		}

		if queued {
			status.Restoring = append(status.Restoring, indice)
		}

		if deleting {
			status.Deleting = append(status.Deleting, indice)
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
		queued := stringInList(indiceStatus.Deleting, indice)
		if stringInList(indiceStatus.Ready, indice) && !queued {
			deleteQueue.Push(&Node{indice})
			deleting = true
		}
	}

	return deleting, nil
}

func stringInList(list []string, a string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
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
