# Backblaze B2 integration

## Configuration

The configuration parameters are as follows:

- `location` (required): the bucket name (e.g, `b2://my-bucket-name`)
- `bucketID` (required): the bucket ID
- `key_id` (required): the Backblaze Key ID
- `application_id` (required): the Backblaze Application Key

## Examples

```
$ plakar store add b2 b2://myBucket bucketId=YOUR_BUCKET_ID key_id=YOUR_KEY_ID application_id=YOUR_APPLICATION_ID

$ plakar at @b2 create
```
