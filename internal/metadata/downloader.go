package metadata

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"gowin32/internal"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

const definitionAddress string = "https://api.nuget.org/v3/index.json"
const nugetName string = "microsoft.windows.sdk.win32metadata"

func DownloadMetadata(metadataFileName string) {
	baseAddress := getBaseAddress()
	versionsResponse, err := queryGet(fmt.Sprintf("%s%s/index.json", baseAddress, nugetName))
	internal.PanicOnError(err)
	versions, err := parse[map[string][]string](versionsResponse)
	internal.PanicOnError(err)
	orderedVersions := make([]*version.Version, len(versions["versions"]))
	for i, versionString := range versions["versions"] {
		version, err := version.NewVersion(versionString)
		if err != nil {
			panic(fmt.Errorf("error parsing version: %s", versionString))
		}

		orderedVersions[i] = version
	}

	sort.Sort(version.Collection(orderedVersions))
	version := orderedVersions[len(orderedVersions)-1].Original()
	nugetBytes, err := queryGet(fmt.Sprintf("%s%s/%s/%s.%s.nupkg", baseAddress, nugetName, version, nugetName, version))
	internal.PanicOnError(err)

	bytesReader := bytes.NewReader(nugetBytes)
	nuget, err := zip.NewReader(bytesReader, int64(bytesReader.Len()))
	internal.PanicOnError(err)
	for _, file := range nuget.File {
		if filepath.Ext(file.Name) == ".winmd" {
			reader, err := file.Open()
			internal.PanicOnError(err)
			metadataBytes, err := io.ReadAll(reader)
			internal.PanicOnError(err)
			os.WriteFile(metadataFileName, metadataBytes, 0644)
			return
		}
	}
}

func getBaseAddress() string {
	response, err := queryGet(definitionAddress)
	internal.PanicOnError(err)
	nugetIndex, err := parse[nugetIndex](response)
	internal.PanicOnError(err)

	for _, resource := range nugetIndex.Resources {
		if strings.Contains(resource.Type, "PackageBaseAddress") {
			return resource.Id
		}
	}

	return ""
}

func parse[T interface{}](source []byte) (T, error) {
	var parsedBody T
	err := json.Unmarshal(source, &parsedBody)
	return parsedBody, err
}

func queryGet(url string) ([]byte, error) {
	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, nil // ToDo: Handle error
	}

	return io.ReadAll(response.Body)
}

type nugetIndex struct {
	Resources []nugetResource `json:"resources"`
}

type nugetResource struct {
	Id   string `json:"@id"`
	Type string `json:"@type"`
}
