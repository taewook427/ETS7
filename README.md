# ETS7 v1.1.0

> Kutil gen7, Essential Tools Series 7

## TestLog

- TestLog는 테스트와 코드 번호를 기록합니다. TestLog records tests and code numbers.
- `index,timestamp,content` 형태의 레코드를 관리합니다. Manages records in the format `index,timestamp,content`.
- 실행 시 모든 레코드를 표시하고 자동으로 다음 번호를 준비합니다. Displays all records and automatically prepares the next number.

## SyncIt

- 파일 해시값을 계산하고 두 폴더의 내용물을 동기화합니다. Calculates file hash ​​and synchronizes two folders.
- 다음 해시 알고리즘 지원: `crc32, md5, sha1, sha2-256, sha2-512, sha3-256, sha3-512` Supports following algorithms: `crc32, md5, sha1, sha2-256, sha2-512, sha3-256, sha3-512`
- 폴더 동기화는 src의 데이터는 수정하지 않고 dst의 내용물과 구조를 동기화합니다. Synchronizes the contents and structure of dst without modifying src.

| flag | value | info |
| :-- | :-- | :-- |
| -src | dirpath | Set source dir |
| -dst | dirpath | Set destination dir |
| -crc | | Use CRC32 to compare file |
| | filepath | Default arg is for file hash |

## RepairIt

- 지정된 폴더에 포함된 파일의 해시값과 패리티를 저장합니다. Stores the hash values ​​and parity of files contained in a specified folder.
- 하드웨어 비트 로트를 탐지하고 복구합니다. Detects and recovers hardware bit rots.
- 패리티 데이터는 다음 구조입니다: `[size 8B][CRCs][Parity][meta-crc 4B]` The parity data has the following structure: `[size 8B][CRCs][Parity][meta-crc 4B]`

| flag | value | info |
| :-- | :-- | :-- |
| -prt | dirpath | Set parity storage |
| -check | | Enable readonly mode |
| | dirpath | Default arg is for target dir |

## PicIt

- 데이터를 암호화하여 사진 픽셀의 LSB에 숨깁니다. Encrypts data and hides it in the LSB of the photo pixels.
- JPG, PNG, WEBP 형식의 주형 사진을 설정하면 PNG 결과를 내보냅니다. Gets template photo in JPG, PNG, or WEBP, and exports a PNG result.
- 기본적으로 2개의 서브픽셀에 1바이트를 인코딩하며 subtle 모드는 4개의 서브픽셀을 사용합니다. 1 byte is encoded for 2 subpixels by default, and subtle mode uses 4 subpixels.

| config | info |
| :-- | :-- |
| size | Fyne GUI size |
| format | Output file extension |
| key | Default encryption key |

## ImgConv

- 다음 이미지 형식을 지원합니다: `jpg, png, webp, ico` Supports the following image formats: `jpg, png, webp, ico`
- 회전, 대칭, 병합, 크기와 형식 변환이 가능한 이미지 편집도구 입니다. An image editing tool capable of rotation, mirroring, merging, and resizing/format conversion.
- 흑백, 색반전, 색강화, 색약화 효과를 낼 수 있습니다. Can produce black and white, color inversion, color enhancement, and color weakening effects.
- 회전 효과는 육십분법으로 돌아갈 각도를 받으며 기본값은 90입니다. Rotation effect takes a rotation angle in degrees, with a default value of 90.
- 색강조 효과는 실수를 받으며 0은 색 지우기, 1은 변화없음으로 작용합니다. Color highlight effect takes a real number, with 0 acting as color erase and 1 as no change.

## FileConv

- 대량의 파일을 한 번에 변환합니다. Converts a large number of files at once.
- 다음과 같은 이미지로의 형식변환을 지원합니다: `jpg, png, webp` Supports conversion to the following image formats: `jpg, png, webp`
- 다음과 같은 파일의 병합을 지원합니다: `image, pdf` Supports merging the following files: `image, pdf`

## TZConv

- 인자로 폴더나 *.tar.zst 파일을 받아 압축하거나 해제합니다. Get dirpath or *.tar.zst file by argument, compress or decompress it.

## Build Executable

This application uses Go programming language. [Install Go](https://go.dev/) to build yourself, or download pre-built release binary. It takes few minutes to download and build GUI version.

windows cli
```bat
go mod init example.com
go mod tidy
go build -ldflags="-s -w" -trimpath -o code.exe code.go
```

linux/mac cli
```bash
go mod init example.com
go mod tidy
go build -ldflags="-s -w" -trimpath -o code code.go
```

windows gui
```bat
go mod init example.com
go mod tidy
go build -ldflags="-H windowsgui -s -w" -trimpath -o code.exe code.go
```

linux/mac gui
```bash
go mod init example.com
go mod tidy
go build -ldflags="-s -w" -trimpath -o code code.go
```

fyne2 GUI requires C compiler and X11 environment. Selection dialog requires Zenity. check and install following packages before build.
```bash
gcc --version
sudo apt install zenity
sudo apt-get install pkg-config libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```
