# Gaia pre-store-enricher
Enrich the event with data pre storing it into the DB.

**For instance:** Adding information such as "incoming_time", so even if the event does not have any time information Gaia consumer will be able at least to retrieve the information according to the arrival time of the event into Gaia servers.

Here is example for event post enrichment (the **"gaia"** json part is the enrichement info):
```javascript
{
  "gaia":
    {"incoming_time":"2016-04-03T12:42:40Z"},
  //event properties
  "key":"value" 
}
```
