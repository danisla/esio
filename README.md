# Elasticsearch Snapshot Index Orchestrator

- Accepts requests to restore indices from snapshot repository.
- Incoming requests are queued for restore and queue is processed every 5 seconds.
- Requests for additional index restorations are queued while other recoveries are in progress.
- Available index snapshots are listed via the ES [Snapshot/Restore API](https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html).
- Ongoing recoveries and online indices are obtained via the ES [Index API](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html).
