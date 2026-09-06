package storage

import (
    "context"
    "fmt"
    "net/http"
    "io"
    "os"
    "encoding/json"
    "strings"

    "github.com/PlakarKorp/kloset/connectors/storage"
    "github.com/PlakarKorp/kloset/location"
    "github.com/PlakarKorp/kloset/objects"
)

type Store struct {
    apiUrl string
    bucketName string
    authToken string
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
    //os.Stderr.WriteString(fmt.Sprintf("%v", storeConfig))
    var bucketName string
    if value, ok := storeConfig["location"]; !ok {
        return nil, fmt.Errorf("missing location")
    } else {
        // TODO: error if location does not start with "b2://" ??
        bucketName = strings.TrimPrefix(value, "b2://")
    }

    var keyId string
    if value, ok := storeConfig["key_id"]; !ok {
        return nil, fmt.Errorf("missing key_id")
    } else {
        keyId = value
    }

    var applicationKey string
    if value, ok := storeConfig["application_key"]; !ok {
        return nil, fmt.Errorf("missing application_key")
    } else {
        applicationKey = value
    }

    client := &http.Client{}
    req, err := http.NewRequest("GET", "https://api.backblazeb2.com/b2api/v4/b2_authorize_account", nil)
    if err != nil {
        return nil, fmt.Errorf("Unable to connect to backblaze api:", err)
    }

    req.SetBasicAuth(keyId, applicationKey)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Unable to authenticate using backblaze api:", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Unable to read auth response to backblaze api:", err)
    }

    var jsonRes map[string]interface{}

    err = json.Unmarshal(body, &jsonRes)
    if err != nil {
        return nil, fmt.Errorf("Unable to parse JSON response from backblaze api:", err)
    }

    // TODO: check that we have all the capabilities necessaries to peform the next operations
    // but maybe just display them, store them, because we might only need to push, not to pull
    authToken := jsonRes["authorizationToken"].(string)

    // TODO: better error checking scheme ?
    apiUrl := jsonRes["apiInfo"].(map[string]interface{})["storageApi"].(map[string]interface{})["apiUrl"].(string)

    os.Stderr.WriteString(bucketName)
    os.Stderr.WriteString("\n")
    os.Stderr.WriteString(authToken)
    os.Stderr.WriteString("\n")
    os.Stderr.WriteString(apiUrl)
    os.Stderr.WriteString("\n")

    return &Store{
        apiUrl: apiUrl,
        bucketName: bucketName,
        authToken: authToken,
    }, nil
}

func (s *Store) Origin() string { return "" }
func (s *Store) Root() string { return "" }
func (s *Store) Type() string { return "b2" }

func (s *Store) Size(ctx context.Context) (int64, error) {
    return 0, fmt.Errorf(">> SIZE")
}

func (s *Store) Ping(ctx context.Context) error {
    return fmt.Errorf(">> PING")
}

func (s *Store) Flags() location.Flags { return 0 }

func (s *Store) Create(ctx context.Context, config []byte) error {
    return fmt.Errorf(">> CREATE")
}

func (s *Store) Delete(ctx context.Context, res storage.StorageResource, mac objects.MAC) error {
    return fmt.Errorf(">> DELETE")
}

func (s *Store) Get(ctx context.Context, res storage.StorageResource, mac objects.MAC, rg *storage.Range) (io.ReadCloser, error) {
    return nil, fmt.Errorf(">> GET")
}

func (s *Store) Put(ctx context.Context, res storage.StorageResource, mac objects.MAC, rd io.Reader) (int64, error) {
    return -1, fmt.Errorf(">> PUT")
}

func (s *Store) List(ctx context.Context, res storage.StorageResource) ([]objects.MAC, error) {
    return nil, fmt.Errorf(">> LIST")
}

func (s *Store) Mode(ctx context.Context) (storage.Mode, error) {
    // TODO: based on capabilities
    return 0, fmt.Errorf(">> MODE")
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
    return nil, fmt.Errorf(">> OPEN")
}

func (s *Store) Close(ctx context.Context) error {
    return nil
}
