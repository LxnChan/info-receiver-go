package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/go-sql-driver/mysql"
)

// ClientInfo 客户端信息结构体
type ClientInfo struct {
	// 主机名
	Name    string `json:"Name"`
	CPU     string `json:"CPU"`
	RAM     string `json:"RAM"`
	Disk    string `json:"Disk"`
	SN      string `json:"SN"`
	MAC     string `json:"MAC"`
	IP      string `json:"IP"`
	// 客户端版本
	UpVer   string `json:"up_ver"`
	Comment string `json:"comment"`
	// 网络类型，判断其实不准
	Network string `json:"Network"`
}

// Database 数据库连接结构体
type Database struct {
	conn *sql.DB
}

// NewDatabase 创建新的数据库连接
func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	return &Database{conn: db}, nil
}

// CreateTable 创建数据表
func (db *Database) CreateTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS client_info (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255),
		cpu VARCHAR(255),
		ram VARCHAR(255),
		disk VARCHAR(255),
		sn VARCHAR(255),
		mac VARCHAR(255),
		ip VARCHAR(255),
		up_ver VARCHAR(255),
		comment TEXT,
		network VARCHAR(255),
        post_at TIMESTAMP NULL DEFAULT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_mac (mac)
	)`
	
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("创建数据表失败: %v", err)
	}

    // 兼容已存在的表，确保存在 post_at 字段
	// 注释掉了，mysql8不支持这个
    //if _, err := db.conn.Exec("ALTER TABLE client_info ADD COLUMN IF NOT EXISTS post_at TIMESTAMP NULL DEFAULT NULL"); err != nil {
        // 某些 MySQL 版本不支持 IF NOT EXISTS，这里忽略 "Duplicate column" 错误
    //    if !isDuplicateColumnError(err) {
    //        return fmt.Errorf("添加 post_at 字段失败: %v", err)
    //    }
    //}
	
    // 创建变更记录表
    changes := `
    CREATE TABLE IF NOT EXISTS client_changes (
        id INT AUTO_INCREMENT PRIMARY KEY,
        client_id INT NOT NULL,
        change_type VARCHAR(16) NOT NULL, -- insert/update
        name VARCHAR(255),
        cpu VARCHAR(255),
        ram VARCHAR(255),
        disk VARCHAR(255),
        sn VARCHAR(255),
        mac VARCHAR(255),
        ip VARCHAR(255),
        up_ver VARCHAR(255),
        comment TEXT,
        network VARCHAR(255),
        changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_client_id (client_id),
        INDEX idx_change_mac (mac)
    )`

    if _, err := db.conn.Exec(changes); err != nil {
        return fmt.Errorf("创建变更记录表失败: %v", err)
    }

	return nil
}

// CheckExistingRecord 检查是否存在相同的MAC或SN记录
func (db *Database) CheckExistingRecord(info *ClientInfo) (int, error) {
	query := `
	SELECT id FROM client_info 
	WHERE (mac = ? AND mac != '') OR (sn = ? AND sn != '')
	LIMIT 1`
	
	var id int
	err := db.conn.QueryRow(query, info.MAC, info.SN).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // 没有找到重复记录
		}
		return 0, fmt.Errorf("查询重复记录失败: %v", err)
	}
	
	return id, nil
}

// InsertOrUpdateClientInfo 插入或更新客户端信息，返回结果类型：insert/update/nochange
func (db *Database) InsertOrUpdateClientInfo(info *ClientInfo) (string, error) {
	// 检查是否存在重复记录
	existingId, err := db.CheckExistingRecord(info)
	if err != nil {
        return "", err
	}
	
    if existingId > 0 {
        // 读取现有记录用于比较
        var cur ClientInfo
        sel := `SELECT name, cpu, ram, disk, sn, mac, ip, up_ver, comment, network FROM client_info WHERE id = ?`
        if err := db.conn.QueryRow(sel, existingId).Scan(
            &cur.Name, &cur.CPU, &cur.RAM, &cur.Disk, &cur.SN, &cur.MAC, &cur.IP, &cur.UpVer, &cur.Comment, &cur.Network,
        ); err != nil {
            return "", fmt.Errorf("读取现有数据失败: %v", err)
        }

        isSame := cur.Name == info.Name && cur.CPU == info.CPU && cur.RAM == info.RAM && cur.Disk == info.Disk &&
            cur.SN == info.SN && cur.MAC == info.MAC && cur.IP == info.IP && cur.UpVer == info.UpVer &&
            cur.Comment == info.Comment && cur.Network == info.Network

        if isSame {
            // 无变化，仅更新 post_at
            onlyPostAt := `UPDATE client_info SET post_at = CURRENT_TIMESTAMP WHERE id = ?`
            if _, err := db.conn.Exec(onlyPostAt, existingId); err != nil {
                return "", fmt.Errorf("更新post_at失败: %v", err)
            }
            return "nochange", nil
        }

        // 有变化：更新字段并刷新 post_at
        query := `
        UPDATE client_info SET 
            name = ?, cpu = ?, ram = ?, disk = ?, 
            sn = ?, mac = ?, ip = ?, up_ver = ?, 
            comment = ?, network = ?, post_at = CURRENT_TIMESTAMP
        WHERE id = ?`
        
        if _, err := db.conn.Exec(query, info.Name, info.CPU, info.RAM, info.Disk,
            info.SN, info.MAC, info.IP, info.UpVer, info.Comment, info.Network, existingId); err != nil {
            return "", fmt.Errorf("更新数据失败: %v", err)
        }
        // 写入变更记录
        if err := db.logChange(existingId, "update", info); err != nil {
            return "", err
        }
        return "update", nil
	} else {
		// 插入新记录
        query := `
        INSERT INTO client_info (name, cpu, ram, disk, sn, mac, ip, up_ver, comment, network, post_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
		
        res, err := db.conn.Exec(query, info.Name, info.CPU, info.RAM, info.Disk,
            info.SN, info.MAC, info.IP, info.UpVer, info.Comment, info.Network)
		
		if err != nil {
            return "", fmt.Errorf("插入数据失败: %v", err)
		}
        // 获取新插入的ID并记录变更
        newId, _ := res.LastInsertId()
        if newId > 0 {
            if err := db.logChange(int(newId), "insert", info); err != nil {
                return "", err
            }
        }
		
        return "insert", nil
	}
}

// logChange 将变更记录写入client_changes表
func (db *Database) logChange(clientID int, changeType string, info *ClientInfo) error {
    query := `
    INSERT INTO client_changes (
        client_id, change_type, name, cpu, ram, disk, sn, mac, ip, up_ver, comment, network
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    _, err := db.conn.Exec(query, clientID, changeType, info.Name, info.CPU, info.RAM, info.Disk,
        info.SN, info.MAC, info.IP, info.UpVer, info.Comment, info.Network)
    if err != nil {
        return fmt.Errorf("记录变更失败: %v", err)
    }
    return nil
}

// Close 关闭数据库连接
func (db *Database) Close() error {
	return db.conn.Close()
}

// handleClientData 处理客户端数据POST请求
func handleClientData(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置响应头
		w.Header().Set("Content-Type", "application/json")
		
		// 只允许POST请求
		if r.Method != http.MethodPost {
			http.Error(w, "只允许POST请求", http.StatusMethodNotAllowed)
			return
		}
		
		// 解析JSON数据
		var clientInfo ClientInfo
		if err := json.NewDecoder(r.Body).Decode(&clientInfo); err != nil {
			log.Printf("JSON解析失败: %v", err)
			http.Error(w, "JSON数据格式错误", http.StatusBadRequest)
			return
		}
		
		// 所有字段都是可选的，不需要验证
		
        // 插入或更新数据库
        result, err := db.InsertOrUpdateClientInfo(&clientInfo)
		if err != nil {
			log.Printf("数据库操作失败: %v", err)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
			return
		}
		
		// 返回成功响应
		var message string
        switch result {
        case "insert":
            message = "数据已成功保存"
        case "update":
            message = "数据已成功更新"
        case "nochange":
            message = "数据无变化，已记录上报时间"
        default:
            message = "操作已完成"
        }
		
		response := map[string]string{
			"status": "success",
			"message": message,
		}
		
		json.NewEncoder(w).Encode(response)
		
		// 记录操作类型
        switch result {
        case "insert":
            log.Printf("成功保存新客户端数据: %s (%s) - MAC: %s, SN: %s",
                clientInfo.Name, clientInfo.IP, clientInfo.MAC, clientInfo.SN)
        case "update":
            log.Printf("成功更新客户端数据: %s (%s) - MAC: %s, SN: %s",
                clientInfo.Name, clientInfo.IP, clientInfo.MAC, clientInfo.SN)
        case "nochange":
            log.Printf("客户端数据无改变: %s (%s) - MAC: %s, SN: %s",
                clientInfo.Name, clientInfo.IP, clientInfo.MAC, clientInfo.SN)
        }
	}
}

// isDuplicateColumnError 判断是否为重复列错误
func isDuplicateColumnError(err error) bool {
    if err == nil {
        return false
    }
    // MySQL 报错文本包含 "Duplicate column name"
    return strings.Contains(err.Error(), "Duplicate column") || strings.Contains(err.Error(), "Duplicate column name")
}

func main() {
	// 定义命令行参数
	var (
		dsn     = flag.String("dsn", "", "数据库连接字符串 (必需)")
		logDir  = flag.String("log-dir", "", "日志目录 (可选，不指定则不输出日志)")
		port    = flag.String("port", "8080", "服务器端口")
	)
	flag.Parse()
	
	// 检查必需的DSN参数
	if *dsn == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定数据库连接字符串\n")
		fmt.Fprintf(os.Stderr, "使用方法: %s -dsn \"user:password@tcp(host:port)/dbname\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "示例: %s -dsn \"root:password@tcp(localhost:3306)/goup\"\n", os.Args[0])
		os.Exit(1)
	}
	
	// 设置日志
	setupLogging(*logDir)
	
	// 连接数据库
	db, err := NewDatabase(*dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()
	
	// 创建数据表
	if err := db.CreateTable(); err != nil {
		log.Fatalf("创建数据表失败: %v", err)
	}
	
	log.Println("数据库连接成功，数据表已创建")
	
	// 创建路由
	router := mux.NewRouter()
	
	// 添加健康检查端点
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"message": "服务运行正常",
		})
	}).Methods("GET")
	
	// 添加客户端数据接收端点
	router.HandleFunc("/api/client", handleClientData(db)).Methods("POST")
	
	// 启动服务器
	log.Printf("服务器启动在端口 %s", *port)
	log.Printf("健康检查: http://localhost:%s/health", *port)
	log.Printf("客户端数据接口: http://localhost:%s/api/client", *port)
	
	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// setupLogging 设置日志输出
func setupLogging(logDir string) {
	if logDir == "" {
		// 不输出日志
		log.SetOutput(os.Stderr) // 只输出到stderr，不输出到文件
		return
	}
	
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
		os.Exit(1)
	}
	
	// 创建日志文件
	logFile := filepath.Join(logDir, "goup-server.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建日志文件失败: %v\n", err)
		os.Exit(1)
	}
	
	// 设置日志输出到文件
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	fmt.Printf("日志将输出到: %s\n", logFile)
}
