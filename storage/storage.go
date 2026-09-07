package storage

import (
    "context"
    "fmt"
    "net/http"
    "io"
    "encoding/hex"
    "encoding/json"
    "strings"
    "crypto/sha1"
    "errors"

    "github.com/PlakarKorp/kloset/connectors/storage"
    "github.com/PlakarKorp/kloset/location"
    "github.com/PlakarKorp/kloset/objects"
)

type Store struct {
    apiUrl string
    bucketName string
    // TODO: this may be simplified with only the bucket name we can find the bucket ID
    bucketID string
    authToken string
}

func init() {
    storage.Register("b2", 0, NewStore)
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
    var bucketName string
    if value, ok := storeConfig["location"]; !ok {
        return nil, fmt.Errorf("missing location")
    } else {
        // TODO: error if location does not start with "b2://" ??
        bucketName = strings.TrimPrefix(value, "b2://")
    }

    var bucketID string
    if value, ok := storeConfig["bucketID"]; !ok {
        return nil, fmt.Errorf("missing bucketID")
    } else {
        bucketID = value
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
        return nil, fmt.Errorf("Unable to connect to backblaze api: %w", err)
    }

    req.SetBasicAuth(keyId, applicationKey)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Unable to authenticate using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("Unable to authenticate using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Unable to read auth response to backblaze api: %w", err)
    }

    var jsonRes map[string]interface{}

    err = json.Unmarshal(body, &jsonRes)
    if err != nil {
        return nil, fmt.Errorf("Unable to parse JSON response from backblaze api: %w", err)
    }

    // TODO: check that we have all the capabilities necessaries to peform the next operations
    // but maybe just display them, store them, because we might only need to push, not to pull
    authToken := jsonRes["authorizationToken"].(string)

    // TODO: better error checking scheme ?
    apiUrl := jsonRes["apiInfo"].(map[string]interface{})["storageApi"].(map[string]interface{})["apiUrl"].(string)

    return &Store{
        apiUrl: apiUrl,
        bucketName: bucketName,
        bucketID: bucketID,
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

func (s *Store) getUploadUrl() (string, string, error) {
    client := &http.Client{}

    // NOTE: the backblaze API say this request should be a GET, but it seems that 
    // the Go http package does not send the body if we make the request a GET.
    // What can we do ?
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/b2api/v4/b2_get_upload_url", s.apiUrl), strings.NewReader(fmt.Sprintf("{\"bucketId\":\"%s\"}", s.bucketID)))
    req.Header.Add("Authorization", s.authToken)

    resp, err := client.Do(req)
    if err != nil {
        return "", "", fmt.Errorf("Unable to get upload url using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return "", "", fmt.Errorf("Unable to get upload url using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", "", fmt.Errorf("Unable to read response to backblaze api: %w", err)
    }

    var jsonRes map[string]interface{}

    err = json.Unmarshal(body, &jsonRes)
    if err != nil {
        return "", "", fmt.Errorf("Unable to parse JSON response from backblaze api: %w", err)
    }

    uploadUrl := jsonRes["uploadUrl"].(string)
    uploadAuthorizationToken := jsonRes["authorizationToken"].(string)

    return uploadUrl, uploadAuthorizationToken, nil
}

func (s *Store) Create(ctx context.Context, config []byte) error {
    // TODO: check if the bucket properly exists
    uploadUrl, uploadAuthorizationToken, err := s.getUploadUrl()
    if err != nil {
        return err
    }

    h := sha1.New()
    h.Write(config)
    hash := h.Sum(nil)

    client := &http.Client{}
    req, err := http.NewRequest("POST", uploadUrl, strings.NewReader(string(config)))
    req.Header.Add("Authorization", uploadAuthorizationToken)
    req.Header.Add("X-Bz-File-Name", "CONFIG")
    // TODO: binary mime type ?
    req.Header.Add("Content-Type", "text/plain")
    req.Header.Add("X-Bz-Content-Sha1", fmt.Sprintf("%x", hash))

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Unable to upload the config file using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("Unable to download the config file using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    return nil
}

func (s *Store) getFileId(needle string) (string, error) {
    // TODO: this is not optimal, we are iterating on ALL the files in the bucket
    // just to find one file id
    // is there a better way to do this ?

    // TODO: multiple call to scan through all the files !!!
    client := &http.Client{}
    // NOTE: the backblaze API say this request should be a GET, but it seems that 
    // the Go http package does not send the body if we make the request a GET.
    // What can we do ?
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/b2api/v4/b2_list_file_names", s.apiUrl), strings.NewReader(fmt.Sprintf("{\"bucketId\":\"%s\"}", s.bucketID)))
    req.Header.Add("Authorization", s.authToken)
    req.Header.Add("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("Unable to list file names url using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("Unable to list file names url using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("Unable to read response to backblaze api: %w", err)
    }

    var jsonRes map[string]interface{}

    err = json.Unmarshal(body, &jsonRes)
    if err != nil {
        return "", fmt.Errorf("Unable to parse JSON response from backblaze api: %w", err)
    }

    for _, file := range jsonRes["files"].([]interface{}) {
        filename := file.(map[string]interface{})["fileName"].(string)
        if (filename == needle) {

            return file.(map[string]interface{})["fileId"].(string), nil
        }
    }
    return "", fmt.Errorf("file not found")
}

func (s *Store) Delete(ctx context.Context, res storage.StorageResource, mac objects.MAC) error {
    prefix, err := getPrefixFromStorageResource(res)
    if err != nil {
        return err
    }

    filename := fmt.Sprintf("%s%016x", prefix, mac)
    fileId, err := s.getFileId(filename)
    if err != nil {
        return err
    }

    client := &http.Client{}
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/b2api/v4/b2_delete_file_version", s.apiUrl), strings.NewReader(fmt.Sprintf("{\"fileName\":\"%s\", \"fileId\":\"%s\"}", filename, fileId)))
    req.Header.Add("Authorization", s.authToken)
    req.Header.Add("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Unable to delete file using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        // TODO: maybe it's not a big deal if the file was not found ??
        // at least log it and do not stop the whole process
        return fmt.Errorf("Unable to delete file using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    return nil
}

func (s *Store) Get(ctx context.Context, res storage.StorageResource, mac objects.MAC, rg *storage.Range) (io.ReadCloser, error) {
    prefix, err := getPrefixFromStorageResource(res)
    if err != nil {
        return nil, err
    }

    filename := fmt.Sprintf("%s%016x", prefix, mac)

    client := &http.Client{}
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/file/%s/%s", s.apiUrl, s.bucketName, filename), nil)
    req.Header.Add("Authorization", s.authToken)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Unable to get file using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("Unable to get file using backblaze api. Status code: %v", resp.StatusCode)
    }

    return resp.Body, nil
}

func (s *Store) Put(ctx context.Context, res storage.StorageResource, mac objects.MAC, rd io.Reader) (int64, error) {

    prefix, err := getPrefixFromStorageResource(res)
    if err != nil {
        return -1, err
    }

    content, err := io.ReadAll(rd)
    if err != nil {
        return -1, fmt.Errorf("error while reading put content: %w", err)
    }

    uploadUrl, uploadAuthorizationToken, err := s.getUploadUrl()
    if err != nil {
        return -1, err
    }

    // TODO: maybe we could stream the content of the file here to avoid getting it all into memory all at once ?
    h := sha1.New()
    h.Write(content)
    hash := h.Sum(nil)

    client := &http.Client{}
    req, err := http.NewRequest("POST", uploadUrl, strings.NewReader(string(content)))
    req.Header.Add("Authorization", uploadAuthorizationToken)
    req.Header.Add("X-Bz-File-Name", fmt.Sprintf("%s%016x", prefix, mac))
    // TODO: binary mime type ?
    req.Header.Add("Content-Type", "text/plain")
    req.Header.Add("X-Bz-Content-Sha1", fmt.Sprintf("%x", hash))

    resp, err := client.Do(req)
    if err != nil {
        return -1, fmt.Errorf("Unable to upload the config file using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return -1, fmt.Errorf("Unable to upload the config file using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()
    return (int64)(len(content)), nil
}

func getPrefixFromStorageResource(res storage.StorageResource) (string, error) {
    switch res {
    case storage.StorageResourcePackfile:
        return "packfiles/", nil
    case storage.StorageResourceState:
        return "states/", nil
    case storage.StorageResourceLock:
        return "locks/", nil
    default:
        return "", errors.ErrUnsupported
    }
}

func (s *Store) List(ctx context.Context, res storage.StorageResource) ([]objects.MAC, error) {
    prefix, err := getPrefixFromStorageResource(res)
    if err != nil {
        return nil, err
    }

    // TODO: multiple call to scan through all the files !!!
    client := &http.Client{}
    // NOTE: the backblaze API say this request should be a GET, but it seems that 
    // the Go http package does not send the body if we make the request a GET.
    // What can we do ?
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/b2api/v4/b2_list_file_names", s.apiUrl), strings.NewReader(fmt.Sprintf("{\"bucketId\":\"%s\"}", s.bucketID)))
    req.Header.Add("Authorization", s.authToken)
    req.Header.Add("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Unable to list file names url using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("Unable to list file names url using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Unable to read response to backblaze api: %w", err)
    }

    var jsonRes map[string]interface{}

    err = json.Unmarshal(body, &jsonRes)
    if err != nil {
        return nil, fmt.Errorf("Unable to parse JSON response from backblaze api: %w", err)
    }

    ret := make([]objects.MAC, 0)
    for _, file := range jsonRes["files"].([]interface{}) {
        filename := file.(map[string]interface{})["fileName"].(string)
        if strings.HasPrefix(filename, prefix) {
            realFilename := strings.TrimPrefix(filename, prefix)
            t, err := hex.DecodeString(realFilename)
            if err != nil {
                return nil, fmt.Errorf("Unable decode hex filename: %w", err)
            }
            ret = append(ret, objects.MAC(t))
        }
    }

    return ret, nil
}

func (s *Store) Mode(ctx context.Context) (storage.Mode, error) {
    // TODO: based on capabilities
    return 0, fmt.Errorf(">> MODE NOT IMPLEMENTED")
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
    client := &http.Client{}

    // NOTE: the backblaze API say this request should be a GET, but it seems that 
    // the Go http package does not send the body if we make the request a GET.
    // What can we do ?
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/file/%s/CONFIG", s.apiUrl, s.bucketName), nil)
    req.Header.Add("Authorization", s.authToken)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Unable to download the config file using backblaze api: %w", err)
    }
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("Unable to download the config file using backblaze api. Status code: %v", resp.StatusCode)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Unable to read auth response to backblaze api: %w", err)
    }

    return body, nil
}

func (s *Store) Close(ctx context.Context) error {
    return nil
}
