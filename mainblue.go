package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	tableName = "untitled_table_1"
)

type TodoBlue struct {
	Id         int        `json:"id"`
	Password   string     `json:"password"`
	Balance    int        `json:"balance"`
	Username   string     `json:"username"`
	TimeDonate *time.Time `json:"timeDonate,omitempty"`
}

var db *gorm.DB

type TodoBlueCreation struct {
	Id       int    `json:"id"`
	Password string `json:"password"`
	Balance  int    `json:"balance"`
	Username string `json:"username"`
}

type Account struct {
	Id       int
	Username string
	Password string
	Balance  int
}

func (TodoBlueCreation) TableName() string {
	return tableName
}

func (Account) TableName() string {
	return tableName
}

func (TodoBlue) TableName() string {
	return tableName
}

func main() {
	dsn := os.Getenv("DB_STR_BLUE")
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}

	r := gin.Default()

	// Endpoint cho tạo tài khoản
	r.POST("/create-account", createAccountHandler)

	// Endpoint cho đăng nhập tài khoản
	r.POST("/login", loginHandler)

	// Endpoint cho nạp số dư vào tài khoản
	r.POST("/update-balance", updateBalanceHandler)

	// Endpoint cho kiểm tra số dư
	r.GET("/check-balance/:id", checkBalanceHandler)

	// Endpoint cho chuyển tiền giữa hai tài khoản
	r.POST("/transfer", transferBalanceHandler)

	err = r.Run(":2609")
	if err != nil {
		panic(err)
	}
}

// Tạo tài khoản mới
func CreateAccount(id int, username, password string, balance int) TodoBlueCreation {
	newAccount := TodoBlueCreation{
		Id:       id,
		Username: username,
		Password: password,
		Balance:  balance,
	}

	return newAccount
}

func createAccountHandler(c *gin.Context) {
	var newAccountRequest TodoBlueCreation

	if err := c.ShouldBindJSON(&newAccountRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//todo: tao them 1 bien de check username cua newAccountRequest va bien do co trung nhau hay ko

	var oldAccount TodoBlue
	if err := db.Where("username = ?", newAccountRequest.Username).First(&oldAccount).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Tài khoản đã tồn tại"})
		return
	}

	newAccount := CreateAccount(newAccountRequest.Id, newAccountRequest.Username, newAccountRequest.Password, newAccountRequest.Balance)

	//todo: luu tai khoan

	//todo:check neu username da ton tai roi thi khong the tao tai khoan cung 1 username trong db

	if err := db.Create(&newAccountRequest).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	db.Save(&newAccountRequest)

	c.JSON(http.StatusOK, newAccount)
}

// Đăng nhập vào tài khoản
func LoginAccount(username, password string) (*TodoBlue, error) {
	var account TodoBlue
	err := db.Where("username = ?", username).First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("Tài khoản không tồn tại")
		}
		return nil, err
	}

	if account.Password != password {
		return nil, fmt.Errorf("Mật khẩu không chính xác")
	}
	return &account, nil
}

func loginHandler(c *gin.Context) {
	var loginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := LoginAccount(loginRequest.Username, loginRequest.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đăng nhập thành công",
		"account": account,
	})
}

// Nạp số dư vào tài khoản
func UpdateBalance(db *gorm.DB, username string, amount int) error {
	var account Account

	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("Tài khoản không tồn tại")
		}
		return err
	}

	account.Balance += amount

	log.Printf("Số dư mới: %d\n", account.Balance)

	if err := db.Updates(&account).Error; err != nil {
		return err
	}
	return nil
}

func updateBalanceHandler(c *gin.Context) {
	var request struct {
		UserName string `json:"username"`
		Amount   int    `json:"amount"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := UpdateBalance(db, request.UserName, request.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Nạp tiền thành công"})
}

// Kiểm tra số dư
func CheckBalance(db *gorm.DB, username string) (int, error) {
	var account Account

	if err := db.Where("username = ? ", username).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("Tai khoan khong ton tai")
		}
		return 0, err
	}

	return account.Balance, nil
}

func checkBalanceHandler(c *gin.Context) {
	var request struct {
		UserName string `json:"username"`
	}

	if err := c.ShouldBindQuery(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance, err := CheckBalance(db, request.UserName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Số dư hiện tại của tài khoản là: %d", balance)})

}

// Chuyển tiền giữa các tài khoản
func Bank(db *gorm.DB, fromUserID int, toUserID int, amount int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var fromAccount, toAccount Account

		if err := tx.First(&fromAccount, fromUserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("Tài khoản không tồn tại")
			}
			return err
		}

		if err := tx.First(&toAccount, toUserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("Tài khoản không tồn tại")
			}
			return err
		}

		if fromAccount.Balance < amount {
			return fmt.Errorf("Tài khoản không đủ số dư")
		}

		fromAccount.Balance -= amount
		toAccount.Balance += amount

		if err := tx.Save(&fromAccount).Error; err != nil {
			return err
		}

		if err := tx.Save(&toAccount).Error; err != nil {
			return err
		}

		return nil
	})
}

func transferBalanceHandler(c *gin.Context) {
	var request struct {
		FromUserID int `json:"from_user_id"`
		ToUserID   int `json:"to_user_id"`
		Amount     int `json:"amount"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := Bank(db, request.FromUserID, request.ToUserID, request.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chuyển tiền thành công"})
}

//todo: ham kiem tra lich su nap tai khoan

//func CheckTimeDonate(db *gorm.DB, userID int) (time.Time, error) {
//	var account TodoBlue
//
//	if err := db.First(&account, userID).Error; err != nil {
//		if errors.Is(err, gorm.ErrRecordNotFound) {
//			return time.Now(), fmt.Errorf("Tài khoản không tồn tại")
//		}
//		return 0, err
//	}
//
//	return account.Balance, nil
//
//}
