package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

// Local types for monitoring server only
type MonitoringFile struct {
	Success bool                 `json:"success"`
	Result  MonitoringFileResult `json:"result"`
}

type MonitoringFileResult struct {
	Count   int                        `json:"count"`
	Results []MonitoringFileResultItem `json:"results"`
}

type MonitoringFileResultItem struct {
	Id               string                 `json:"id"`
	Title            string                 `json:"title"`
	MetadataModified string                 `json:"metadata_modified"`
	NumResources     int                    `json:"num_resources"`
	Organization     MonitoringOrganization `json:"organization"`
	Tags             []MonitoringTag        `json:"tags"`
}

type MonitoringOrganization struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

type MonitoringTag struct {
	DisplayName string `json:"display_name"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"`
}

type MonitoringData struct {
	LastUpdate string
	Datasets   []DatasetInfo
}

type DatasetInfo struct {
	Name             string
	ID               string
	Organization     string
	LastModified     string
	LastModifiedTime time.Time
	NumResources     int
	Tags             []string
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>DataSoup Monitoring</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .dataset-link { color: #0066cc; text-decoration: none; }
        .dataset-link:hover { text-decoration: underline; }
        .tags { font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🍲 DataSoup Monitoring</h1>
        <p><strong>Last Update:</strong> {{.LastUpdate}}</p>
        <p><strong>Total Datasets:</strong> {{len .Datasets}}</p>
    </div>
    
    <table>
        <thead>
            <tr>
                <th>Dataset Name</th>
                <th>Organization</th>
                <th>Last Modified</th>
                <th>Resources</th>
                <th>Tags</th>
            </tr>
        </thead>
        <tbody>
            {{range .Datasets}}
            <tr>
                <td>
                    <a href="https://data.gov.il/dataset/{{.ID}}" class="dataset-link" target="_blank">
                        {{.Name}}
                    </a>
                </td>
                <td>{{.Organization}}</td>
                <td>{{.LastModified}}</td>
                <td>{{.NumResources}}</td>
                <td class="tags">{{range $i, $tag := .Tags}}{{if $i}}, {{end}}{{$tag}}{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
`

func loadMonitoringData() (*MonitoringData, error) {
	// Get file info for last update time
	fileInfo, err := os.Stat("data/packagedata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Read and parse the JSON file
	data, err := os.ReadFile("data/packagedata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var datafile MonitoringFile
	err = json.Unmarshal(data, &datafile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Convert to monitoring data
	var datasets []DatasetInfo
	for _, pkg := range datafile.Result.Results {
		lastModified, err := time.Parse("2006-01-02T15:04:05.000000", pkg.MetadataModified)
		if err != nil {
			log.Printf("Error parsing time for %s: %v", pkg.Id, err)
			continue
		}

		var tags []string
		for _, tag := range pkg.Tags {
			tags = append(tags, tag.DisplayName)
		}

		datasets = append(datasets, DatasetInfo{
			Name:             pkg.Title,
			ID:               pkg.Id,
			Organization:     pkg.Organization.Title,
			LastModified:     lastModified.Format("2006-01-02 15:04"),
			LastModifiedTime: lastModified,
			NumResources:     pkg.NumResources,
			Tags:             tags,
		})
	}

	// Sort by last modified time (most recent first)
	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].LastModifiedTime.After(datasets[j].LastModifiedTime)
	})

	return &MonitoringData{
		LastUpdate: fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		Datasets:   datasets,
	}, nil
}

func monitoringHandler(w http.ResponseWriter, r *http.Request) {
	data, err := loadMonitoringData()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading data: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("monitoring").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", monitoringHandler)

	fmt.Println("DataSoup monitoring server starting on :8080")
	fmt.Println("Visit http://localhost:8080 to view the monitoring dashboard")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
