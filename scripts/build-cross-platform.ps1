# Perfgo 交叉编译脚本 (PowerShell版本)
# 支持构建多个平台的二进制文件

param(
    [string]$Version = "v1.0.0"
)

$ErrorActionPreference = "Stop"

# 版本信息
$BuildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
try {
    $GitCommit = git rev-parse HEAD 2>$null
    if ($GitCommit) {
        $GitCommit = $GitCommit.Substring(0, 7)
    } else {
        $GitCommit = ""
    }
} catch {
    $GitCommit = ""
}

# 创建输出目录
if (!(Test-Path "dist")) {
    New-Item -ItemType Directory -Path "dist" | Out-Null
}

Write-Host "开始构建 Perfgo $Version (构建时间: $BuildTime)"

# 定义要构建的平台
$Platforms = @(
    @{ OS = "windows"; ARCH = "amd64"; Ext = ".exe" },
    @{ OS = "linux"; ARCH = "amd64"; Ext = "" },
    @{ OS = "linux"; ARCH = "arm64"; Ext = "" },
    @{ OS = "darwin"; ARCH = "amd64"; Ext = "" },
    @{ OS = "darwin"; ARCH = "arm64"; Ext = "" }
)

# 遍历平台进行构建
foreach ($Platform in $Platforms) {
    $GOOS = $Platform.OS
    $GOARCH = $Platform.ARCH
    
    Write-Host "正在构建 $GOOS/$GOARCH..."
    
    $OutputName = "dist/perfgo-$Version-$GOOS-$GOARCH$($Platform.Ext)"
    
    # 设置环境变量并构建
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "0"
    
    $LdFlags = "-X main.Version=$Version -X main.BuildTime=$BuildTime"
    if ($GitCommit) {
        $LdFlags += " -X main.GitCommit=$GitCommit"
    }
    
    go build -ldflags="$LdFlags" -o $OutputName .
    
    Write-Host "  -> 已生成: $(Split-Path $OutputName -Leaf)"
    
    # 清除环境变量
    Remove-Item Env:\GOOS
    Remove-Item Env:\GOARCH
    Remove-Item Env:\CGO_ENABLED
}

Write-Host ""
Write-Host "交叉编译完成！生成的文件位于 dist/ 目录下："
Get-ChildItem -Path "dist/"