# wordpress-debug-scanner
"Security tool to find exposed debug.log files in WordPress websites"
# WordPress Debug.log Scanner

A security tool to identify exposed debug.log files and misconfigurations in WordPress websites.

## 🚨 Disclaimer
This tool is for **educational purposes and authorized security testing only**. Always obtain proper permission before scanning any website.

## 📥 Download
Go to download the latest `wpdebugfinder.exe`
Go Lang Must be installed 
Yype this command  to make Go lang to EXE for windows 
go build -o wpdebugfinder.exe wpdebugfinder.go

## 🛠 Features
- Scan for exposed debug.log files
- Detect WP_DEBUG configuration
- Multi-target scanning capability
- Concurrent scanning with configurable threads
- Results export to file

## 📷 Usage Examples

### Single Target Scan
wpdebugfinder.exe -u example.com

### Multi-Target Scan  
wpdebugfinder.exe -l targets.txt -t 20 -o results.txt



