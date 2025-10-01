package script

import (
	"os"
	"path/filepath"
	"runtime"
)

// IsCNBCloudEnvironment 判断是否是 CNB 云开发环境
// IsCNBCloudEnvironment 判断当前是否运行在 CNB 云环境中
// 通过检查多个 CNB 特有的环境变量来确定是否处于 CNB 云环境
// 如果任何一个 CNB 特有的环境变量存在且符合条件，则返回 true
func IsCNBCloudEnvironment() bool {
	// 检查多个 CNB 特有的环境变量
	return os.Getenv("CNB") == "true" ||
		os.Getenv("CNB_WEB_PROTOCOL") != "" ||
		os.Getenv("CNB_API_ENDPOINT") != "" ||
		os.Getenv("CNB_GROUP_SLUG") != "" ||
		os.Getenv("CNB_TOKEN_USER_NAME") == "cnb"
}

// IsCIEnvironment 判断是否是 CI 环境
/**
 * 判断当前是否处于 CI (持续集成) 环境中
 * @return bool 如果是 CI 环境返回 true，否则返回 false
 */
func IsCIEnvironment() bool {
	// 检查常见的 CI 环境变量
	// 检查通用 CI 环境变量
	return os.Getenv("CI") == "true" ||
		// 检查 GitLab CI 环境变量
		os.Getenv("GITLAB_CI") == "true" ||
		// 检查 GitHub Actions 环境变量
		os.Getenv("GITHUB_ACTIONS") == "true" ||
		// 检查 Travis CI 环境变量
		os.Getenv("TRAVIS") == "true" ||
		// 检查 Jenkins CI 环境变量
		os.Getenv("JENKINS_URL") != "" ||
		// 检查 CNB 云环境（也是 CI 环境）
		IsCNBCloudEnvironment() // CNB 也是 CI 环境
}

// GetDataPath 获取数据路径，根据环境自动选择
// GetDataPath 函数用于获取 Beancount 数据文件的存储路径
// 该函数会根据不同的运行环境（开发环境、CI环境、本地环境等）返回不同的路径
// 优先级顺序为：环境变量 > CNB云环境 > 其他CI环境 > 本地环境
func GetDataPath() string {
	// 首先检查环境变量覆盖
	// 如果设置了 BEANCOUNT_DATA_PATH 环境变量，则直接使用该路径
	if envPath := os.Getenv("BEANCOUNT_DATA_PATH"); envPath != "" {
		return envPath
	}

	// CNB 云开发环境使用固定路径
	// 如果是 CNB 云环境，则返回预定义的固定路径
	if IsCNBCloudEnvironment() {
		return "/workspace/data/beancount"
	}

	// 其他 CI 环境（非 CNB）
	// 如果是其他 CI 环境，则返回预设的 CI 数据路径
	if IsCIEnvironment() {
		// 可以根据需要设置其他 CI 环境的默认路径
		return "/data/beancount"
	}

	// 本地开发环境，根据操作系统选择
	// 在本地开发环境中，根据操作系统返回不同的路径格式
	if runtime.GOOS == "windows" {
		return ".\\data\\beancount" // Windows 使用反斜杠路径
	} else {
		// Linux/Mac 本地开发
		return "./data/beancount" // Linux/Mac 使用正斜杠路径
	}
}

// GetCNBEnvironmentInfo 获取 CNB 环境信息（用于调试和日志）
// GetCNBEnvironmentInfo 获取CNB相关环境变量信息的函数
// 返回一个包含CNB相关环境变量及其值的map
func GetCNBEnvironmentInfo() map[string]string {
	// 创建一个map用于存储环境变量信息
	info := make(map[string]string)

	// 定义需要获取的CNB环境变量列表
	cnbEnvVars := []string{
		"CNB", "CNB_WEB_PROTOCOL", "CNB_WEB_HOST",
		"CNB_WEB_ENDPOINT", "CNB_API_ENDPOINT",
		"CNB_GROUP_SLUG", "CNB_GROUP_SLUG_LOWERCASE",
		"CNB_EVENT", "CNB_EVENT_URL", "CNB_BRANCH",
		"CNB_BRANCH_SHA", "CNB_DEFAULT_BRANCH",
		"CNB_TOKEN_USER_NAME", "CI",
	}

	// 遍历环境变量列表，获取每个环境变量的值
	for _, envVar := range cnbEnvVars {
		// 使用os.Getenv获取环境变量值，如果值不为空则存入map
		if value := os.Getenv(envVar); value != "" {
			info[envVar] = value
		}
	}

	// 返回包含所有有效环境变量的map
	return info
}

// GetPlatformAwarePath 将路径转换为当前平台合适的格式
// GetPlatformAwarePath 根据不同的操作系统环境返回合适的路径格式
// 参数:
//   - path: 输入的路径字符串
// 返回值:
//   - string: 处理后的路径字符串
func GetPlatformAwarePath(path string) string {
	// 判断是否为Windows操作系统，且不是CNBCloud环境，也不是CI环境
	if runtime.GOOS == "windows" && !IsCNBCloudEnvironment() && !IsCIEnvironment() {
		// 如果是Windows环境，将路径从Unix格式转换为Windows格式
		return filepath.FromSlash(path)
	}
	// 其他情况直接返回原始路径
	return path
}

// GetServerConfigFilePath 获取服务器配置文件的完整路径
// 该函数返回配置文件的路径，如果发生错误则返回错误信息
// 返回值:
//   - string: 配置文件的完整路径
//   - error: 错误信息，如果获取路径成功则为nil
func GetServerConfigFilePath() string {
	// 获取基础数据路径
	currentPath, err := os.Getwd()
	if err != nil {
		// 错误处理：返回默认路径或记录错误
		LogSystemError("Failed to get current working directory: " + err.Error())
		return "./config/config.json" // 返回相对路径作为备选
	}
	// 将基础路径与配置文件路径拼接，返回完整的配置文件路径
	return filepath.Join(currentPath, "config", "config.json")
}

// GetServerWhiteListFilePath 用于获取服务器白名单文件的完整路径
// 该函数返回一个文件路径字符串和一个可能的错误信息
// 函数通过组合基础路径和特定文件名来构建完整的文件路径
func GetServerWhiteListFilePath() (string) {
	// 获取基础数据路径
	currentPath, err := os.Getwd()
	if err != nil {
		LogSystemError("Failed to get current working directory: " + err.Error())
		return "./config/white_list.json"
	}
	// 使用filepath.Join函数安全地拼接路径，确保跨平台兼容性
	// 返回完整的配置文件路径，文件名为white_list.json
	return filepath.Join(currentPath, "config", "white_list.json")
}

// GetServerLedgerConfigFilePath 获取服务器账本配置文件的路径
// 函数返回一个字符串表示账本配置文件的完整路径，以及可能的错误信息
// 返回值:
//   - string: 账本配置文件的完整路径
//   - error: 如果发生错误则返回错误信息，否则返回nil
func GetServerLedgerConfigFilePath() (string) {
	// 获取基础数据路径
	basePath := GetDataPath()
	// 将基础路径与账本配置文件名拼接，形成完整的文件路径
	return filepath.Join(basePath, "ledger_config.json")
}

// GetTemplateLedgerConfigDirPath 函数用于获取模板账本配置文件的目录路径
// 该函数接收无参数，返回一个字符串和一个错误值
// 返回的字符串表示模板账本配置文件的目录路径，错误值表示过程中可能出现的错误
func GetTemplateLedgerConfigDirPath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		LogSystemError("Failed to get current working directory: " + err.Error())
		return "./template"
	}
	return filepath.Join(currentPath, "template")
}

// GetLedgerConfigDocument 根据数据路径获取账本配置文件的完整路径
// 参数:
//   dataPath: string - 数据文件的目录路径
// 返回值:
//   string - 配置文件的完整路径，路径格式为 "dataPath/.beancount-gs"
func GetLedgerConfigDocument(dataPath string) string {
	return filepath.Join(dataPath, ".beancount-gs") // 使用filepath.Join函数安全地拼接路径，确保跨平台兼容性
}

func GetCompatibleLedgerConfigDocument(dataPath string) string {
	return filepath.Join(dataPath, ".beancount-ns")
}

// GetLedgerTransactionsTemplateFilePath 函数用于获取账本交易模板文件的路径
// 参数:
//   - dataPath: 数据文件的根目录路径
// 返回值:
//   - string: 交易模板文件的完整路径
func GetLedgerTransactionsTemplateFilePath(dataPath string) string {
	// 使用filepath.Join函数安全地拼接路径，确保跨平台兼容性
	// 拼接dataPath、".beancount-gs"目录和"transaction_template.json"文件名
	return filepath.Join(dataPath, ".beancount-gs", "transaction_template.json")
}

func GetLedgerAccountTypeFilePath(dataPath string) string {
	return filepath.Join(dataPath, ".beancount-gs", "account_type.json")
}

func GetLedgerCurrenciesFilePath(dataPath string) string {
	return filepath.Join(dataPath, ".beancount-gs", "currency.json")
}

func GetLedgerPriceFilePath(dataPath string) string {
	return filepath.Join(dataPath, "price", "prices.bean")
}

// GetLedgerMonthsFilePath 根据数据路径获取账本月份文件的完整路径
// 该函数用于生成账本月份数据文件的存储路径
// @param dataPath 数据存储的基础路径
// @return string 返回月份文件的完整路径，格式为：基础路径/month/months.bean
func GetLedgerMonthsFilePath(dataPath string) string {
	return filepath.Join(dataPath, "month", "months.bean")
}

func GetLedgerMonthFilePath(dataPath string, month string) string {
	return filepath.Join(dataPath, "month", month+".bean")
}

func GetLedgerIndexFilePath(dataPath string) string {
	path := filepath.Join(dataPath, "index.bean")
	LogInfo(dataPath, path)
	return path
}

func GetLedgerIncludesFilePath(dataPath string) string {
	path := filepath.Join(dataPath, "includes.bean")
	LogInfo(dataPath, path)
	return path
}

func GetLedgerEventsFilePath(dataPath string) string {
	path := filepath.Join(dataPath, "event", "events.bean")
	LogInfo(dataPath, path)
	return path
}
