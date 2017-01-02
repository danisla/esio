package restapi

import (
	"container/list"
	"crypto/tls"
	"encoding/json"
	"log"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"time"

	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"

	strftime "github.com/hhkbp2/go-strftime"
	elastic "gopkg.in/olivere/elastic.v2"

	"github.com/danisla/esio/models"
	"github.com/danisla/esio/restapi/operations"
	"github.com/danisla/esio/restapi/operations/health"
	"github.com/danisla/esio/restapi/operations/index"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target .. --name Esio --spec ../swagger.yml

type SnapshotResponse struct {
	Snapshots []Snapshot `json:snapshots`
}

type Snapshot struct {
	Snapshot 	string 		`json:snapshot`
	VersionId int 	 		`json:version_id`
	Indices   []string 	`json:indices`
	State     string    `json:state`
}

var myFlags = struct {
	EsHost string `long:"es-host" description:"Elasticsearch Host [$ES_HOST]"`
	MaxRestore int `long:"max-restore" description:"Maximum number of indices allowed to restore at once, default is 1 [$MAX_RESTORE]"`
	IndexResolution string `long:"resolution" description:"Resolution of indices being restored (day, month, year) [$INDEX_RESOLUTION]"`
	RepoPattern string `long:"repo-pattern" description:"Snapshot repo pattern (repo/snap/index), ex: logs-%y/logs-%y-%m-%d/logs-v1-%y-%m-%d, [$REPO_PATTERN]"`
}{}

func configureFlags(api *operations.EsioAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }

	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
    swag.CommandLineOptionsGroup{
			ShortDescription: "ESIO Flags",
			LongDescription: "",
			Options: &myFlags,
		},
  }
}

func configureAPI(api *operations.EsioAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// s.api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	if myFlags.EsHost == "" {
		if os.Getenv("ES_HOST") != "" {
			myFlags.EsHost = os.Getenv("ES_HOST")
		} else {
			panic("No es-host flag or ES_HOST env provided.")
		}
	}

	api.IndexGetStartEndHandler = index.GetStartEndHandlerFunc(func(params index.GetStartEndParams) middleware.Responder {
 		var msg = ""
		utc, _ := time.LoadLocation("UTC")

		// Parse the start time
		if params.Start < 0 {
			msg = "Start time must be greater than 0"
			return index.NewGetStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}
		start := time.Unix(params.Start, 0)
		start = start.In(utc)
		start_fmt := strftime.Format("%Y-%m-%d %H:%M:%S UTC", start)

		// Parse the end time
		if params.End < 0 {
			msg = "End time must be greater than 0"
			return index.NewGetStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}
		end := time.Unix(params.End, 0)
		end = end.In(utc)
		end_fmt := strftime.Format("%Y-%m-%d %H:%M:%S UTC", end)

		// Time range must be valid
		if params.Start >= params.End {
			msg = "Start time must be less than end time"
			return index.NewGetStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// Index resolution override
		var indexResolution = myFlags.IndexResolution
		if params.Resolution != nil && *params.Resolution != "" {
			indexResolution = *params.Resolution
		}

		// Repo pattern override
		var repoPattern = myFlags.RepoPattern
		if params.RepoPattern != nil && *params.RepoPattern != "" {
			repoPattern = *params.RepoPattern
		}

		// Look for indices in given range.
		indices := makeIndexListFromRange(start, end, indexResolution, repoPattern)

		// Iterate through list and print its contents.
		var allPass = true
		for e := indices.Front(); e != nil; e = e.Next() {
			allPass = allPass && validateSnapshotIndex(e.Value.(string))
		}

		if allPass {
			log.Println("All indices found")
		} else {
			log.Println("Missing indices")
		}

		msg = "No indices are available for restore in given [" + start_fmt + ", " + end_fmt + "] range."
		return index.NewGetStartEndRequestRangeNotSatisfiable().WithPayload(&models.Error{Message: &msg})
	})

	api.IndexPostStartEndHandler = index.PostStartEndHandlerFunc(func(params index.PostStartEndParams) middleware.Responder {
		return middleware.NotImplemented("operation index.PostStartEnd has not yet been implemented")
	})

	api.HealthGetHealthzHandler = health.GetHealthzHandlerFunc(func(params health.GetHealthzParams) middleware.Responder {
		var status = "OK"
		var message = "Healthy"

		client, err := elastic.NewClient(
			elastic.SetURL(myFlags.EsHost),
			elastic.SetHealthcheck(false),
			elastic.SetSniff(false))
		if err != nil {
		  status = "ERROR"
			message = fmt.Sprintf("%s", err)
		}

		// Get cluster health
		res, err := client.ClusterHealth().Do()
		if err != nil {
			status = "ERROR"
			message = fmt.Sprintf("%s", err)
		}
		if res == nil {
			status = "ERROR"
			message = "Error connecting to cluster"
		}

		if status != "ERROR" {
			return health.NewGetHealthzOK().WithPayload(&models.Healthz{Status: &status, Message: &message})
		} else {
			return health.NewGetHealthzDefault(503).WithPayload(&models.Healthz{Status: &status, Message: &message})
		}
	})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}

// Create a list of indices to be restored from the given start,end range.
// Snapshots are derived from the given repoPattern and discritized at intervals of given indexResolution
func makeIndexListFromRange(start time.Time, end time.Time, indexResolution string, repoPattern string) *list.List {
	var l = list.New()
	// starting from start time, make index pattern
	// Increment start time by IndexResolution
	// Add next interval to list until time exceedes end time.

	var t = start

	for t.Before(end) {
		l.PushBack(strftime.Format(repoPattern, t))
		switch indexResolution {
			case "day": t = t.AddDate(0,0,1)
			case "month": t = t.AddDate(0,1,0)
			case "year": t = t.AddDate(1,0,0)
			default: panic("Unhandled IndexResolution switch case: " + indexResolution)
		}
	}
	return l
}

// Verifies each index pattern in given list is found on the ES cluster.
func validateSnapshotIndex(repoPattern string) bool {
	url := fmt.Sprintf("%s/_snapshot/%s", myFlags.EsHost, path.Dir(repoPattern))
	target := path.Base(repoPattern)

	log.Println("Checking snapshot: " + url + " for index: " + target)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return false
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return false
	}

	defer resp.Body.Close()

	var snap SnapshotResponse

	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		log.Println(err)
		return false
	}

	if len(snap.Snapshots) == 0 {
		return false
	}

	var indices = snap.Snapshots[0].Indices
	sort.Strings(indices)

	i := sort.Search(len(indices),
	    func(i int) bool { return indices[i] >= target })
	if i < len(indices) && indices[i] == target {
			return true
	}

	log.Println("Could not locate index in repo: ", target)

	return false
}
