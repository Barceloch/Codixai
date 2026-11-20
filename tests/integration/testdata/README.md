# SARIF 2.1.0 schema fixture

`sarif-schema-2.1.0.json` is a verbatim copy of the OASIS Standard JSON schema
for SARIF 2.1.0, retrieved once during WU3 and committed as test data so schema
validation is reproducible without network access.

- Source: <https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/schemas/sarif-schema-2.1.0.json>
- Canonical `$id`: <https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json>
- Retrieved: 2026-11-19
- SHA-256: `ad6db49878699b091f3eeb765b6e29e92a34bad4da88664d000c923b549c3a25`
- Dialect: JSON Schema draft-07

The SchemaStore copy referenced by the documents (`https://json.schemastore.org/sarif-2.1.0.json`)
was also retrieved and compared. After removing descriptions, titles, defaults
and examples and normalizing `const` to a one-value `enum`, the remaining
differences are redundant `properties` entries inside `anyOf`/`oneOf` branches;
they do not change which documents are accepted.

The schema is data only. The tool never reads it at runtime and never fetches
a schema over the network.
