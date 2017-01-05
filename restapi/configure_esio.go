package restapi

import (
	"crypto/tls"
	// "encoding/json"
	"log"
	"fmt"
	"net/http"
	"os"
	"path"

	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"

	elastic "gopkg.in/olivere/elastic.v2"

	"github.com/danisla/esio/models"
	"github.com/danisla/esio/restapi/operations"
	"github.com/danisla/esio/restapi/operations/health"
	"github.com/danisla/esio/restapi/operations/index"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target .. --name Esio --spec ../swagger.yml

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

	// Initialize the restore queue
	initQueues()
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

		start, end, err := parseTimeRange(params.Start, params.End)
		if err != nil {
			msg := fmt.Sprintf("%s", err)
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
		indices, err := makeIndexListFromRange(start, end, indexResolution, repoPattern)
		if err != nil {
			msg = fmt.Sprintf("Could not make index range: %s", err)
			return index.NewGetStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// Iterate through list and validate each index.
		var allPass = true
		for _, i := range indices {
			passed, err := validateSnapshotIndex(i)
			if err != nil {
				msg = fmt.Sprintf("Error validating index: %s: %s", i, err)
				return index.NewGetStartEndRequestRangeNotSatisfiable().WithPayload(&models.Error{Message: &msg})
			}
			allPass = allPass && passed
		}

		// Create the IndexStatus data structure
		indiceStatus, err := makeIndexStatus(indices)
		if err != nil {
			msg = fmt.Sprintf("Error comparing online indices with snapshots list: %s", err)
			return index.NewGetStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// See if all requested indices are Ready
		var allReady = true
		var allPending = true
		var restoringOrPending = true
		var target = ""
		for _, i := range indices {
			target = path.Base(i)
			allReady = allReady && stringInList(indiceStatus.Ready, target)
			allPending = allPending && stringInList(indiceStatus.Pending, target)
			restoringOrPending = restoringOrPending && (stringInList(indiceStatus.Restoring, target) || stringInList(indiceStatus.Pending, target))
		}

		if allReady {
			return index.NewGetStartEndOK().WithPayload(&indiceStatus)
		}

		if allPending {
			return index.NewGetStartEndNotFound().WithPayload(&indiceStatus)
		}

		// See if some of the requested indices are Ready
		if restoringOrPending {
			return index.NewGetStartEndPartialContent().WithPayload(&indiceStatus)
		}

		// DEBUG
		// b, err := json.Marshal(indiceStatus)
    // if err != nil {
    //   fmt.Println(err)
    // } else {
		// 	log.Println(string(b))
		// }

		msg = fmt.Sprintf("Error processing current indice status.")
		return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
	})

	api.IndexPostStartEndHandler = index.PostStartEndHandlerFunc(func(params index.PostStartEndParams) middleware.Responder {
		var msg = ""

		start, end, err := parseTimeRange(params.Start, params.End)
		if err != nil {
			msg := fmt.Sprintf("%s", err)
			return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
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
		indices, err := makeIndexListFromRange(start, end, indexResolution, repoPattern)
		if err != nil {
			msg = fmt.Sprintf("Could not make index range: %s", err)
			return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// Iterate through list and validate each index.
		var allPass = true
		for _, i := range indices {
			passed, err := validateSnapshotIndex(i)
			if err != nil {
				msg = fmt.Sprintf("Error validating index: %s: %s", i, err)
				return index.NewPostStartEndRequestRangeNotSatisfiable().WithPayload(&models.Error{Message: &msg})
			}
			allPass = allPass && passed
		}

		// Create the IndexStatus data structure
		indiceStatus, err := makeIndexStatus(indices)
		if err != nil {
			msg = fmt.Sprintf("Error comparing online indices with snapshots list: %s", err)
			return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// See if all requested indices are Ready
		var allReady = true
		var restoreStarted = false
		var target = ""
		for _, i := range indices {
			target = path.Base(i)

			// If index is not ready and is pending, then start restoring it.
			if !stringInList(indiceStatus.Ready, target) && stringInList(indiceStatus.Pending, target) {
				allReady = false

				// Queue for restore
				restoreQueue.Push(&Node{target})

				log.Println(fmt.Sprintf("Index queued for restore: %s", i))

				restoreStarted = true
			}
		}

		// Rebuild the indice status
		newIndiceStatus, err := makeIndexStatus(indices)
		if err != nil {
			msg = fmt.Sprintf("Error comparing online indices with snapshots list: %s", err)
			return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// If all indices are online and ready then we are done.
		if allReady {
			return index.NewPostStartEndOK().WithPayload(&newIndiceStatus)
		}

		if restoreStarted {
			return index.NewPostStartEndAccepted().WithPayload(&newIndiceStatus)
		}

		// If we made it here then some of the indices are restoring
		return index.NewPostStartEndPartialContent().WithPayload(&newIndiceStatus)
	})

	api.IndexDeleteStartEndHandler = index.DeleteStartEndHandlerFunc(func(params index.DeleteStartEndParams) middleware.Responder {
		var msg = ""

		start, end, err := parseTimeRange(params.Start, params.End)
		if err != nil {
			msg := fmt.Sprintf("%s", err)
			return index.NewDeleteStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
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
		indices, err := makeIndexListFromRange(start, end, indexResolution, repoPattern)
		if err != nil {
			msg = fmt.Sprintf("Could not make index range: %s", err)
			return index.NewDeleteStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// Not allowed to delete restoring indices
		for _, indice := range indices {
			if restoreQueue.Contains(indice) {
				msg = fmt.Sprintf("Index in range is currently being restored and cannot be deleted at this time: %s", indice)
				return index.NewDeleteStartEndRequestRangeNotSatisfiable().WithPayload(&models.Error{Message: &msg})
			}
		}

		deleteActive, err := deleteIndices(indices)
		if err != nil {
			msg = fmt.Sprintf("Error deleting index: %s", err)
			return index.NewDeleteStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		// Create the IndexStatus data structure
		indiceStatus, err := makeIndexStatus(indices)
		if err != nil {
			msg = fmt.Sprintf("Error comparing online indices with snapshots list: %s", err)
			return index.NewPostStartEndBadRequest().WithPayload(&models.Error{Message: &msg})
		}

		if deleteActive {
			return index.NewDeleteStartEndAccepted().WithPayload(&indiceStatus)
		}

		return index.NewDeleteStartEndOK().WithPayload(&indiceStatus)

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
