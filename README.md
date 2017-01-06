# Elasticsearch Snapshot Index Orchestrator

RESTful API for orchestrating the restoration of time-based Elasticsearch snapshots.

This API gives you a simple interface for restoring snapshots based on a given time range.

For example, if snapshots are stored in the following _daily_ (repo/snap/index) pattern: `logs-%Y/logs-%Y-%m-%d/logs-v1-%Y-%m-%d` then requests made to the `POST /{start}/{end}` route would trigger a restore of indices in the range that maps to the pattern.

High level of what the API does:

- Accepts requests to restore indices from snapshot repository.
- Incoming requests are queued for restore and queue is processed every 2 seconds.
- Requests for additional index restorations are queued while other recoveries are in progress.
- Available index snapshots are listed via the ES [Snapshot/Restore API](https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html).
- Ongoing recoveries and online indices are obtained via the ES [Index API](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html).

## OpenAPI Spec

The [OpenAPI 2.0](https://github.com/OAI/OpenAPI-Specification) spec is here: [./swagger.yml](./swagger.yml)

## Configuration

```
ESIO Flags:
      --es-host=         Elasticsearch Host [$ES_HOST]
      --max-restore=     Maximum number of indices allowed to restore at once, default is 1 [$MAX_RESTORE]
      --resolution=      Resolution of indices being restored (day, month, year) [$INDEX_RESOLUTION]
      --repo-pattern=    Snapshot repo pattern (repo/snap/index), ex: logs-%y/logs-%y-%m-%d/logs-v1-%y-%m-%d,
                         [$REPO_PATTERN]
```

## Resources

- [go-swagger v0.7.4](https://github.com/go-swagger/go-swagger/tree/0.7.4)
- [OpenAPI v2.0 Spec](https://github.com/OAI/OpenAPI-Specification)
- [Snapshot/Restore API](https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html)
