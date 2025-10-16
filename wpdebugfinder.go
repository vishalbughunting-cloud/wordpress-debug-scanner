package main

import (
    "bufio"
    "crypto/tls"
    "flag"
    "fmt"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
)

type Result struct {
    URL      string
    Status   int
    Size     int64
    Error    error
}

var (
    client *http.Client
    wg     sync.WaitGroup
    results chan Result
)

func init() {
    client = &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },
    }
}

func checkDebugLog(targetURL string) {
    defer wg.Done()

    debugPaths := []string{
        "/wp-content/debug.log",
        "/debug.log",
        "/wp-admin/debug.log",
        "/wp-content/uploads/debug.log",
        "/wp-content/plugins/debug.log",
        "/wp-content/themes/debug.log",
        "/logs/debug.log",
        "/wp-content/logs/debug.log",
        "/../debug.log",
        "/wp-content/../debug.log",
    }

    for _, path := range debugPaths {
        fullURL := strings.TrimSuffix(targetURL, "/") + path
        
        resp, err := client.Head(fullURL)
        if err != nil {
            results <- Result{URL: fullURL, Status: 0, Error: err}
            continue
        }
        resp.Body.Close()

        if resp.StatusCode == 200 {
            resp, err := client.Get(fullURL)
            if err != nil {
                results <- Result{URL: fullURL, Status: resp.StatusCode, Error: err}
                continue
            }
            defer resp.Body.Close()

            scanner := bufio.NewScanner(resp.Body)
            isLikelyLog := false
            lineCount := 0
            
            for scanner.Scan() && lineCount < 5 {
                line := scanner.Text()
                if strings.Contains(line, "PHP") || 
                   strings.Contains(line, "DEBUG") || 
                   strings.Contains(line, "ERROR") || 
                   strings.Contains(line, "Warning") ||
                   strings.Contains(line, "Notice") ||
                   strings.Contains(line, "Fatal") ||
                   strings.Contains(line, "Stack trace") {
                    isLikelyLog = true
                    break
                }
                lineCount++
            }

            results <- Result{
                URL:    fullURL,
                Status: resp.StatusCode,
                Size:   resp.ContentLength,
                Error:  nil,
            }

            if isLikelyLog {
                fmt.Printf("[!] POTENTIAL DEBUG LOG FOUND: %s (Size: %d bytes)\n", fullURL, resp.ContentLength)
            } else if resp.ContentLength > 100 {
                fmt.Printf("[?] Possible debug log (needs verification): %s (Size: %d bytes)\n", fullURL, resp.ContentLength)
            }
        }
    }
}

func checkWPConfig(targetURL string) {
    defer wg.Done()

    paths := []string{
        "/wp-config.php",
        "/wp-config.php.bak",
        "/wp-config.php.save",
        "/wp-config.php.old",
        "/wp-config.php.orig",
        "/wp-config.txt",
        "/wp-config.php.txt",
    }

    for _, path := range paths {
        fullURL := strings.TrimSuffix(targetURL, "/") + path
        
        resp, err := client.Get(fullURL)
        if err != nil {
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == 200 {
            scanner := bufio.NewScanner(resp.Body)
            debugEnabled := false
            for scanner.Scan() {
                line := scanner.Text()
                if strings.Contains(line, "WP_DEBUG") && (strings.Contains(line, "true") || strings.Contains(line, "TRUE")) {
                    debugEnabled = true
                    break
                }
            }
            if debugEnabled {
                fmt.Printf("[!] WP_DEBUG ENABLED in: %s\n", fullURL)
            } else if resp.ContentLength > 100 {
                fmt.Printf("[+] Found wp-config file: %s (Size: %d bytes)\n", fullURL, resp.ContentLength)
            }
        }
    }
}

func main() {
    banner := `
    WordPress Debug.log Finder v1.0
    --------------------------------
    [!] For authorized security testing only!
    `
    fmt.Println(banner)

    var (
        target      = flag.String("u", "", "Single target URL")
        targetFile  = flag.String("l", "", "File containing list of URLs")
        threads     = flag.Int("t", 10, "Number of concurrent threads")
        output      = flag.String("o", "", "Output file")
    )
    flag.Parse()

    results = make(chan Result, 1000)

    var targets []string

    if *target != "" {
        if !strings.HasPrefix(*target, "http") {
            *target = "https://" + *target
        }
        targets = append(targets, *target)
    } else if *targetFile != "" {
        file, err := os.Open(*targetFile)
        if err != nil {
            fmt.Printf("Error opening file: %v\n", err)
            os.Exit(1)
        }
        defer file.Close()

        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            url := strings.TrimSpace(scanner.Text())
            if url != "" && !strings.HasPrefix(url, "http") {
                url = "https://" + url
            }
            targets = append(targets, url)
        }
    } else {
        fmt.Println("Usage:")
        fmt.Println("  Single target: wpdebugfinder.exe -u example.com")
        fmt.Println("  Target list:   wpdebugfinder.exe -l targets.txt")
        fmt.Println("  Threads:       wpdebugfinder.exe -t 20 -l targets.txt")
        fmt.Println("  Output:        wpdebugfinder.exe -l targets.txt -o results.txt")
        fmt.Println("\nExamples:")
        fmt.Println("  wpdebugfinder.exe -u https://wordpress-site.com")
        fmt.Println("  wpdebugfinder.exe -l sites.txt -t 15 -o output.txt")
        flag.PrintDefaults()
        os.Exit(1)
    }

    // Start result processor
    go func() {
        var outputFile *os.File
        var err error
        
        if *output != "" {
            outputFile, err = os.Create(*output)
            if err != nil {
                fmt.Printf("Error creating output file: %v\n", err)
                return
            }
            defer outputFile.Close()
        }

        for result := range results {
            if result.Status == 200 {
                msg := fmt.Sprintf("[FOUND] %s - Status: %d, Size: %d bytes\n", result.URL, result.Status, result.Size)
                fmt.Print(msg)
                
                if outputFile != nil {
                    outputFile.WriteString(msg)
                }
            }
        }
    }()

    fmt.Printf("[*] Scanning %d targets for debug.log files...\n\n", len(targets))
    
    semaphore := make(chan struct{}, *threads)

    for _, target := range targets {
        semaphore <- struct{}{}
        wg.Add(2)
        
        go func(t string) {
            checkDebugLog(t)
            <-semaphore
        }(target)
        
        go func(t string) {
            checkWPConfig(t)
            <-semaphore
        }(target)
    }

    wg.Wait()
    close(results)
    
    time.Sleep(1 * time.Second)
    fmt.Println("\n[*] Scan completed!")
}