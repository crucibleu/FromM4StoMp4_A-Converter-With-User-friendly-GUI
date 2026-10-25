## 为什么开发此程序？
bilibili下架了很多视频，之前收藏和缓存的视频均无法播放

![image](https://github.com/mzky/m4s-converter/assets/13345233/ea8bc799-e47d-40ca-bde4-c47193f0e453)

- 收藏的视频建议缓存起来，使用本程序将缓存的音视频m4s文件合并成mp4，方便再次播放。

- 本工具使用GPAC的MP4Box进行音视频合成(v1.5.0之前版本支持FFMpeg)，能够确保合成后的视频质量和同步性。


### 下载后双击执行或通过命令行执行，需要可执行权限
- https://github.com/mzky/m4s-converter/releases/latest


### 图形界面版本 M4S Converter（Windows / Linux）
双击即可使用，不需要命令行。

- **视频库**：自动扫描 B 站缓存目录（电脑客户端和安卓客户端），按合集分组显示封面、分P、时长、分辨率、编码和状态；可以搜索、筛选，单选、按合集全选或按住 Shift 连选
- **只转换选中的视频**，并且可以自由选择输出文件夹和文件命名方式（例如 `合集-UP主/分P.mp4`）
- **转换前预览**：直接播放缓存中的音视频，不需要先转换，也会显示弹幕
- **转换队列**：同时转换 1-4 个视频，显示每个视频的进度、写入速度和剩余时间，失败可以重试
- **内置播放器**：播放输出目录中的视频，记住播放进度、自动播放下一集，支持弹幕叠加（可调透明度、密度、字号）和快捷键；程序内无法解码的视频（例如部分电脑上的 HEVC）可一键用系统播放器打开
- 输出的 mp4 写入标题、UP主、合集等信息，旁边保存 `.xml`（本程序播放用）和 `.ass`（其他播放器用）弹幕文件
- 深色科技风界面，带粒子与过渡动画；动画可在设置中切换为完整 / 精简 / 关闭，播放视频时自动暂停
- 界面随系统语言显示中文或英文；内置 MP4Box，无需另外安装

**下载**：在 [Releases](https://github.com/crucibleu/FromM4StoMp4_A-Converter-With-User-friendly-GUI/releases) 页面下载
- Windows：`*-windows-amd64-setup.exe`（安装版）或 `*-portable.exe`（免安装）。需要 WebView2，Windows 10/11 已自带，安装版会在缺少时自动安装
- Linux：`.deb`（Debian/Ubuntu）或 `.tar.gz`（其他发行版，需要 WebKitGTK 4.1 和 GStreamer，见压缩包内说明）

**自行编译**：图形界面位于 `app/`，基于 [Wails v2](https://wails.io) + Vue 3
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
cd app && wails build                     # Windows
cd app && wails build -tags webkit2_41    # Linux，需要 libgtk-3-dev libwebkit2gtk-4.1-dev
```
推送 `v1.0.0` 这样的标签后，GitHub Actions 会自动构建并发布上述安装包。

> **English:** `app/` is a desktop GUI for Windows and Linux. It browses your Bilibili cache, converts only the videos you pick to any folder with your own naming pattern, previews them before converting, and plays the results with danmaku. Download it from Releases, or build it with `wails build` (add `-tags webkit2_41` on Linux).


### Android手机端合并文件方法 
- 详见：[拷贝文件与合成方法](https://github.com/mzky/m4s-converter/issues/9)


### 除window和linux外，其它环境的依赖工具安装
- 详见：[依赖工具安装](https://github.com/mzky/m4s-converter/wiki/%E4%BE%9D%E8%B5%96%E5%B7%A5%E5%85%B7%E5%AE%89%E8%A3%85)


### 命令行参数
```
# 指定MP4Box路径: ./m4s-converter-amd64.exe -g "D:\GPAC\mp4box.exe" 或 ./m4s-converter-amd64 -g select
 Flags: 
    -h --help         查看帮助信息
    -v --version      查看版本信息
    -a --assoff       关闭自动生成弹幕功能，默认不关闭
    -o --overlay      合成文件时是否覆盖同名视频，默认不覆盖并重命名新文件
    -u --summarize    将未合并的MP3和视频文件放入汇总目录，默认不汇总
    -c --cachepath    自定义视频缓存路径，默认使用bilibili的默认缓存路径
    -g --gpacpath     自定义GPAC的mp4box文件路径,值为select时弹出选择对话框
```


### 验证合成：
```
2023-12-05_16:02:46 [INFO ] 已合成视频文件:中国-美景极致享受-笨蹦崩.mp4
2023-12-05_16:02:46 [INFO ] ==========================================
2023-12-05_16:02:46 [INFO ] 合成的文件:
C:\Users\mzky\Videos\bilibili\output\【获奖学生动画】The Little Poet 小诗人｜CALARTS 2023\【获奖学生动画】The Little Poet 小诗人｜CALARTS 2023-toh糖.mp4
C:\Users\mzky\Videos\bilibili\output\【电影历史_专题片】《影响》致敬中国电影40年【全集】\40年光影记忆-开飞机的巡查司.mp4
C:\Users\mzky\Videos\bilibili\output\“我不是个好导演”，听田壮壮讲述“我和电影的关系”\“我不是个好导演”，听田壮壮讲述“我和电影的关系”-Tatler的朋友们.mp4
C:\Users\mzky\Videos\bilibili\output\【4K8K-世界各地的美景】\中国-美景极致享受-笨蹦崩.mp4
2023-12-05_16:02:46 [INFO ] 已完成本次任务，耗时:5秒
2023-12-05_16:02:46 [INFO ] ==========================================
按回车键退出...
```

- 合成 1.46GB 文件，耗时: 5 秒
- 合成 11.7GB 文件，耗时:38 秒

以上为固态硬盘测试结果, 仅供参考

##
#### 弹幕xml转换为ass使用了此项目
- https://github.com/kafuumi/converter


#### 视频合成使用的工具
- https://gpac.io
- 本程序使用GPAC的MP4Box进行音视频合成，不会对下载的音视频进行转码


#### 本工具无下载视频功能


## 提缺陷和建议

缺陷或建议提交 [issues](https://github.com/mzky/m4s-converter/issues/new/choose) , 最好带上异常视频的URL地址

## Star History

<a href="https://www.star-history.com/#mzky/m4s-converter&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=mzky/m4s-converter&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=mzky/m4s-converter&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=mzky/m4s-converter&type=date&legend=top-left" />
 </picture>
</a>


## ⚠️ **法律声明**  
使用本工具即表示您同意[免责声明](免责声明.md)。  
仅允许转换您本人在视频下架前通过官方客户端合法缓存的内容，且转换结果**严格限于个人备份**，禁止传播与商用。



