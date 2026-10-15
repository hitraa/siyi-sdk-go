package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/media"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to camera web server (port 82)
	client := media.NewClient(protocol.DefaultIP, protocol.DefaultHTTPPort)

	// 1. List date-based image directories
	fmt.Println("Listing photo directories on camera TF card...")
	dirs, err := client.ListDirectories(ctx, models.MediaTypeImages)
	if err != nil {
		log.Fatalf("ListDirectories failed: %v", err)
	}

	fmt.Printf("Found %d photo directories:\n", len(dirs))
	for _, d := range dirs {
		count, err := client.GetMediaCount(ctx, models.MediaTypeImages, d.Path)
		if err != nil {
			fmt.Printf("  Directory: %s (Path: %s)\n", d.Name, d.Path)
		} else {
			fmt.Printf("  Directory: %s (Path: %s) - %d files\n", d.Name, d.Path, count)
		}
	}

	// 2. If directories exist, fetch first batch of files
	if len(dirs) > 0 {
		fmt.Printf("\nFetching files in %s...\n", dirs[0].Path)
		files, err := client.ListFiles(ctx, models.MediaTypeImages, dirs[0].Path, 0, 10)
		if err != nil {
			log.Printf("ListFiles failed: %v", err)
		} else {
			for i, f := range files {
				fmt.Printf("  [%d] %s -> %s\n", i+1, f.Name, f.URL)
			}
		}
	}

	// 3. List video directories
	fmt.Println("\nListing video directories on camera TF card...")
	vDirs, err := client.ListDirectories(ctx, models.MediaTypeVideos)
	if err != nil {
		log.Printf("ListDirectories (video) failed: %v", err)
	} else {
		fmt.Printf("Found %d video directories:\n", len(vDirs))
		for _, vd := range vDirs {
			fmt.Printf("  Video Directory: %s (%s)\n", vd.Name, vd.Path)
		}
	}
}
