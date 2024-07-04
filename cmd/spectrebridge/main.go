package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/spectre-project/spectre-stratum-bridge/src/spectrestratum"
	"gopkg.in/yaml.v2"
)

// BannedWallets là cấu trúc dữ liệu cho danh sách wallet bị cấm
type BannedWallets struct {
	Wallets map[string]bool // map wallet -> true/false (có trong danh sách banned hay không)
}

// IsWalletBanned kiểm tra xem wallet có trong danh sách banned không
func IsWalletBanned(wallet string, banned *BannedWallets) bool {
	_, found := banned.Wallets[wallet]
	return found
}

func main() {
	// Đường dẫn tới file cấu hình YAML
	pwd, _ := os.Getwd()
	fullPath := path.Join(pwd, "config.yaml")
	log.Printf("Loading config @ `%s`", fullPath)

	// Đọc nội dung file cấu hình YAML
	rawCfg, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("Config file not found: %s", err)
		os.Exit(1)
	}

	// Giải mã YAML vào cấu trúc BridgeConfig
	cfg := spectrestratum.BridgeConfig{}
	if err := yaml.Unmarshal(rawCfg, &cfg); err != nil {
		log.Printf("Failed parsing config file: %s", err)
		os.Exit(1)
	}

	// Cấu hình flag.Usage để hiển thị thông tin hữu ích hơn
	flag.Usage = func() {
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(os.Stderr, "  -%v %v\n", f.Name, f.Value)
			fmt.Fprintf(os.Stderr, "    	%v (default \"%v\")\n", f.Usage, f.Value)
		})
	}

	// Định nghĩa các flag và parse từ dòng lệnh
	flag.StringVar(&cfg.StratumPort, "stratum", cfg.StratumPort, "Stratum port to listen on")
	flag.BoolVar(&cfg.PrintStats, "stats", cfg.PrintStats, "True to show periodic stats to console")
	flag.StringVar(&cfg.RPCServer, "spectre", cfg.RPCServer, "Address of the spectred node")
	flag.DurationVar(&cfg.BlockWaitTime, "blockwait", cfg.BlockWaitTime, "Time in ms to wait before manually requesting new block")
	flag.UintVar(&cfg.MinShareDiff, "mindiff", cfg.MinShareDiff, "Minimum share difficulty to accept from miner(s)")
	flag.BoolVar(&cfg.VarDiff, "vardiff", cfg.VarDiff, "True to enable auto-adjusting variable min diff")
	flag.UintVar(&cfg.SharesPerMin, "sharespermin", cfg.SharesPerMin, "Number of shares per minute the vardiff engine should target")
	flag.BoolVar(&cfg.VarDiffStats, "vardiffstats", cfg.VarDiffStats, "Include vardiff stats readout every 10s in log")
	flag.UintVar(&cfg.ExtranonceSize, "extranonce", cfg.ExtranonceSize, "Size in bytes of extranonce")
	flag.StringVar(&cfg.PromPort, "prom", cfg.PromPort, "Address to serve prom stats")
	flag.BoolVar(&cfg.UseLogFile, "log", cfg.UseLogFile, "If true will output errors to log file")
	flag.StringVar(&cfg.HealthCheckPort, "hcp", cfg.HealthCheckPort, "(Rarely used) If defined will expose a health check on /readyz")
	flag.Parse()

	// Log các thông số cấu hình
	log.Println("----------------------------------")
	log.Printf("Initializing bridge")
	log.Printf("\tSpectred:        %s", cfg.RPCServer)
	log.Printf("\tStratum:         %s", cfg.StratumPort)
	log.Printf("\tProm:            %s", cfg.PromPort)
	log.Printf("\tStats:           %t", cfg.PrintStats)
	log.Printf("\tLog:             %t", cfg.UseLogFile)
	log.Printf("\tMin Diff:        %d", cfg.MinShareDiff)
	log.Printf("\tVar Diff:        %t", cfg.VarDiff)
	log.Printf("\tShares per Min:  %d", cfg.SharesPerMin)
	log.Printf("\tVar Diff Stats:  %t", cfg.VarDiffStats)
	log.Printf("\tBlock Wait:      %s", cfg.BlockWaitTime)
	log.Printf("\tExtranonce Size: %d", cfg.ExtranonceSize)
	log.Printf("\tHealth Check:    %s", cfg.HealthCheckPort)
	log.Println("----------------------------------")

	// Đọc danh sách các wallet bị cấm từ file YAML
	banned := &BannedWallets{
		Wallets: make(map[string]bool),
	}
	banedListPath := path.Join(pwd, "baned.yaml")
	if err := readBannedWallets(bannedListPath, banned); err != nil {
		log.Printf("Failed to read banned wallets list: %s", err)
		os.Exit(1)
	}

	// Hàm đọc danh sách wallet bị cấm từ file YAML
	func readBannedWallets(filePath string, banned *BannedWallets) error {
		rawData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(rawData, &banned.Wallets)
		if err != nil {
			return err
		}
		return nil
	}

	// Hàm xử lý kết nối mới
	handleNewConnection := func(wallet string) string {
		if IsWalletBanned(wallet, banned) {
			// Nếu wallet nằm trong danh sách banned, sử dụng wallet mặc định
			fmt.Printf("Connection with banned wallet %s detected. Changing to default wallet.\n", wallet)
			return cfg.DefaultWallet // Thay đổi wallet của kết nối mới thành wallet mặc định
		}
		fmt.Printf("Connection with wallet %s is allowed.\n", wallet)
		return wallet // Giữ nguyên wallet của kết nối mới
	}

	// Lắng nghe và phục vụ kết nối
	log.Println("Listening and serving connections...")
	for {
		// Giả sử có một cơ chế nào đó để lấy wallet của kết nối mới
		newWallet := "example_wallet"
		newWallet = handleNewConnection(newWallet)

		// Gọi hàm ListenAndServe để xử lý kết nối với wallet đã được xử lý
		if err := spectrestratum.ListenAndServe(cfg); err != nil {
			log.Println(err)
		}

		// Chờ một khoảng thời gian và tiếp tục lắng nghe kết nối mới
		time.Sleep(5 * time.Second)
	}
}
