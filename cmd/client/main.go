package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// --- CONFIGURATION ---
const chunkSize = 2 * 1024 * 1024
const lbAddress = "http://localhost:8000"

// --- STRUCTS (For Uploading) ---
type ClientChunk struct {
	ChunkID string `json:"chunk_id"`
	Index   int    `json:"index"`
}
type ClientUploadRequest struct {
	FileName string        `json:"filename"`
	Chunks   []ClientChunk `json:"chunks"`
}
type UploadPlanResponse struct {
	Success    bool                `json:"success"`
	UploadPlan map[string][]string `json:"upload_plan"` // chunkID -> [DN_URL, ...]
}

// --- STRUCTS (For Downloading) ---
type DownloadChunkInfo struct {
	ChunkID   string   `json:"chunk_id"`
	Index     int      `json:"chunk_index"`
	Locations []string `json:"locations"`
}
type DownloadPlanResponse struct {
	Chunks []DownloadChunkInfo `json:"chunks"`
}

// --- SHA1 HELPER ---
func sha1sum(inp []byte) string {
	h := sha1.New()
	h.Write(inp)
	return hex.EncodeToString(h.Sum(nil))
}

// ===================================================================
//
//	UPLOAD LOGIC
//
// ===================================================================

func handleUpload(filePath string) {
	// 1. Break the file into chunks
	log.Printf("Chunking file: %s\n", filePath)
	chunks, data, err := chunkFile(filePath)
	if err != nil {
		log.Fatalf("Failed to chunk file: %v", err)
	}
	log.Printf("File split into %d chunks.\n", len(chunks))

	// 2. Call the Load Balancer to get the upload plan
	log.Println("Contacting load balancer to get upload plan...")
	plan, err := initiateUpload(filepath.Base(filePath), chunks)
	if err != nil {
		log.Fatalf("Failed to get upload plan: %v", err)
	}
	log.Println("Upload plan received.")

	// 3. Follow the plan and upload the data
	log.Println("Starting chunk uploads...")
	uploadChunks(plan, data)

	log.Println("Upload complete!")
}

// chunkFile (Same as before)
func chunkFile(filePath string) ([]ClientChunk, map[string][]byte, error) {
	var chunksMetadata []ClientChunk
	chunkDataMap := make(map[string][]byte)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	buffer := make([]byte, chunkSize)
	index := 0
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunkData := make([]byte, n)
			copy(chunkData, buffer[:n])
			chunkID := sha1sum(chunkData)
			chunksMetadata = append(chunksMetadata, ClientChunk{
				ChunkID: chunkID,
				Index:   index,
			})
			chunkDataMap[chunkID] = chunkData
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return chunksMetadata, chunkDataMap, nil
}

// initiateUpload (Same as before)
func initiateUpload(filename string, chunks []ClientChunk) (map[string][]string, error) {
	reqBody := ClientUploadRequest{FileName: filename, Chunks: chunks}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := http.Post(lbAddress+"/uploadFile", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call load balancer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load balancer returned error: %s", resp.Status)
	}
	var uploadResponse UploadPlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResponse); err != nil {
		return nil, fmt.Errorf("failed to decode upload plan: %w", err)
	}
	return uploadResponse.UploadPlan, nil
}

// uploadChunks (Same as before)
func uploadChunks(uploadPlan map[string][]string, chunkData map[string][]byte) {
	var wg sync.WaitGroup
	for chunkID, locations := range uploadPlan {
		data, ok := chunkData[chunkID]
		if !ok {
			log.Printf("Error: No data found for chunkID %s, skipping", chunkID)
			continue
		}
		wg.Add(1)
		go func(id string, locs []string, d []byte) {
			defer wg.Done()
			uploadChunkToReplicas(id, locs, d)
		}(chunkID, locations, data)
	}
	wg.Wait()
}

// uploadChunkToReplicas (Same as before)
func uploadChunkToReplicas(chunkID string, locations []string, data []byte) {
	var wg sync.WaitGroup
	for _, location := range locations {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			fullURL := fmt.Sprintf("%s/writeChunk/%s", url, chunkID)
			req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(data))
			if err != nil {
				log.Printf("Failed to create request for %s: %v\n", url, err)
				return
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Failed to upload chunk %s to %s: %v\n", chunkID, url, err)
				return
			}
			resp.Body.Close()
		}(location)
	}
	wg.Wait()
}

// ===================================================================
//
//	DOWNLOAD LOGIC (NEW)
//
// ===================================================================

// handleDownload is the main "download" function
func handleDownload(fileName string, saveAs string) {
	// 1. Get the download plan from the LB/Namenode
	log.Println("Contacting load balancer for download plan...")
	plan, err := getDownloadPlan(fileName)
	if err != nil {
		log.Fatalf("Failed to get download plan: %v", err)
	}

	// 2. Download all chunks in parallel
	log.Println("Downloading chunks...")
	// Sort the chunks by their index (0, 1, 2, ...)
	sort.Slice(plan.Chunks, func(i, j int) bool {
		return plan.Chunks[i].Index < plan.Chunks[j].Index
	})
	
	chunkData, err := downloadAllChunks(plan)
	if err != nil {
		log.Fatalf("Failed to download chunks: %v", err)
	}

	// 3. Stitch the file back together
	log.Println("Reassembling file...")
	err = reassembleFile(chunkData, saveAs)
	if err != nil {
		log.Fatalf("Failed to reassemble file: %v", err)
	}

	log.Printf("File successfully downloaded and saved as %s\n", saveAs)
}

// getDownloadPlan calls a *new* LB endpoint
func getDownloadPlan(filename string) (*DownloadPlanResponse, error) {
	reqURL := fmt.Sprintf("%s/get-file-locations?filename=%s", lbAddress, filename)
	log.Printf("lbAddress : %s\n", lbAddress)
	log.Println("reqURL: " + reqURL)
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LB returned error: %s", resp.Status)
	}

	var plan DownloadPlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// downloadAllChunks fetches all chunks (in order) from the Datanodes
func downloadAllChunks(plan *DownloadPlanResponse) ([][]byte, error) {
	// A slice to hold the downloaded data, in the correct order
	chunkData := make([][]byte, len(plan.Chunks))
	
	var wg sync.WaitGroup
	errChan := make(chan error, len(plan.Chunks))

	for i, chunk := range plan.Chunks {
		wg.Add(1)
		go func(c DownloadChunkInfo, index int) {
			defer wg.Done()
			
			// Try the first location
			data, err := downloadChunk(c.Locations[0], c.ChunkID)
			if err != nil {
				// TODO: Add logic to try other locations if the first fails
				errChan <- fmt.Errorf("failed to download chunk %s: %w", c.ChunkID, err)
				return
			}
			chunkData[index] = data
		}(chunk, i)
	}
	
	wg.Wait()
	close(errChan)

	// Check if any errors occurred during download
	if err := <-errChan; err != nil {
		return nil, err
	}
	
	return chunkData, nil
}

// downloadChunk gets one chunk from one Datanode
func downloadChunk(location string, chunkID string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/readChunk/%s", location, chunkID)
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("datanode returned error: %s", resp.Status)
	}
	
	return io.ReadAll(resp.Body)
}

// reassembleFile writes all the chunks to a single output file
func reassembleFile(chunks [][]byte, saveAs string) error {
	file, err := os.Create(saveAs)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, chunk := range chunks {
		_, err := file.Write(chunk)
		if err != nil {
			return err
		}
	}
	return nil
}

// ===================================================================
//
//	MAIN FUNCTION (The "Switch")
//
// ===================================================================

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./client/ [upload|download] [file_path]")
		fmt.Println("  upload [file_to_upload]")
		fmt.Println("  download [filename_to_download] [save_as_path]")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "upload":
		if len(os.Args) < 3 {
			log.Fatal("Usage: go run ./client/ upload [file_to_upload]")
		}
		filePath := os.Args[2]
		handleUpload(filePath)

	case "download":
		if len(os.Args) < 4 {
			log.Fatal("Usage: go run ./client/ download [filename_to_download] [save_as_path]")
		}
		fileName := os.Args[2]
		saveAs := os.Args[3]
		handleDownload(fileName, saveAs)
		
	default:
		log.Fatalf("Unknown command: %s. Use 'upload' or 'download'.", command)
	}
}
