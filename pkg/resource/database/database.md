Use the *clickhousedbops_database* resource to create a database in a ClickHouse instance.

Database engines can be configured with `engine`, `engine_arguments`, and
`engine_settings`. Arguments and setting values are ClickHouse SQL expressions,
so include quotes around string literals. This representation supports database
engines with different signatures without tying the provider to a fixed engine
catalog.

For secret engine arguments or settings, reference a write-only string parameter
such as `{catalog_credential:String}` and provide its value through
`engine_parameters_wo`. The provider safely quotes these string values and
redacts them from logs and errors; write-only values are not stored in state. Bump
`engine_parameters_wo_version` when a secret changes.

Changing the comment or engine configuration recreates the database. **This
destroys its contents.** ClickHouse exposes the engine name through
`system.databases`, but does not expose engine arguments and settings as
structured fields. The provider detects drift of the engine name and retains
configured arguments and settings in state.
