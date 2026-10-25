package common

import (
	"fmt"
	"m4s-converter/conver"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/fatih/color"
	utils "github.com/mzky/utils/common"
	"github.com/sirupsen/logrus"
)

// Result 一次合成任务的结果
type Result struct {
	OutputDir   string
	OutputFiles []string // 相对于 OutputDir 的路径
	SkippedDirs []string // 未缓存完成而跳过的目录
	FailedFiles []string // 合成失败的文件
	Canceled    bool
	ElapsedSecs int64
}

// Synthesis 命令行模式：执行合成任务，打印结果后等待用户按键退出
func (c *Config) Synthesis() {
	res, err := c.Convert()
	if err != nil {
		MessageBox(err.Error())
		c.wait()
	}
	logrus.Print("===========================================")
	if res.SkippedDirs != nil {
		logrus.Print("跳过的目录:\n" + strings.Join(res.SkippedDirs, "\n"))
	}
	if res.OutputFiles != nil {
		logrus.Printf("# 输出目录:\n%s", color.CyanString(res.OutputDir))
		logrus.Printf("# 合成的文件:\n%s", color.CyanString(strings.Join(res.OutputFiles, "\n")))
		// 打开合成文件目录
		go OpenFolder(res.OutputDir)
	} else {
		logrus.Warn("未合成任何文件！")
	}
	logrus.Print("===========================================")
	logrus.Print("已完成合成任务，耗时: ", res.ElapsedSecs, "秒")
	c.wait()
}

// Convert 执行合成任务并返回结果，不会阻塞等待用户输入，也不会退出进程，供命令行与图形界面共用。
// 致命错误（如找不到缓存目录）通过 error 返回；单个视频的失败记录在日志与 Result.FailedFiles 中。
func (c *Config) Convert() (*Result, error) {
	begin := time.Now().Unix()
	res := &Result{}
	logrus.Println("查找缓存目录下可转换的文件...")
	c.progress(StageExtract, 0, 0, "")
	// 查找m4s文件，并转换为mp4和mp3
	if err := filepath.WalkDir(c.CachePath, c.FindM4sFiles); err != nil {
		return nil, fmt.Errorf("查找并转换 m4s 文件异常：%v", err)
	}

	dirs, err := GetCacheDir(c.CachePath) // 缓存根目录模式
	if err != nil {
		return nil, fmt.Errorf("找不到 BiliBili 的缓存目录：%v", err)
	}

	if dirs == nil {
		// 判断非缓存根目录时，验证是否为子目录
		if utils.IsExist(filepath.Join(c.CachePath, conver.VideoInfoSuffix)) ||
			utils.IsExist(filepath.Join(c.CachePath, conver.VideoInfoJson)) {
			dirs = append(dirs, c.CachePath)
		}
	}

	// 合成音视频文件
	c.OutputDir = filepath.Join(c.CachePath, "output")
	res.OutputDir = c.OutputDir
	for i, v := range dirs {
		// 检查是否应该退出
		if c.ShouldExit() {
			logrus.Info("任务已取消")
			res.Canceled = true
			break
		}
		c.progress(StageMerge, i, len(dirs), filepath.Base(v))
		video, audio, e := c.GetAudioAndVideo(v)
		if e != nil {
			logrus.Error("找不到已修复的音频和视频文件:", e)
			continue
		}
		info := filepath.Join(v, conver.VideoInfoJson)
		if !utils.IsExist(info) {
			info = filepath.Join(v, conver.VideoInfoSuffix)
			if !utils.IsExist(info) {
				info = filepath.Join(v, conver.PlayEntryJson)
				if !utils.IsExist(info) {
					continue
				}
			}
		}
		infoStr, e := os.ReadFile(info)
		if e != nil {
			logrus.Error("找不到包含视频信息的info相关文件: ", info)
			continue
		}
		js, e := simplejson.NewJson(infoStr)
		if e != nil {
			logrus.Error("videoInfo相关文件解析失败: ", info)
			continue
		}

		groupTitle := Filter(js.Get("groupTitle").String())
		groupTitle = null2Str(groupTitle, Filter(js.Get("owner_name").String()))

		title := Filter(js.Get("page_data").Get("download_subtitle").String())
		title = null2Str(title, Filter(js.Get("title").String()))

		part := Filter(js.Get("page_data").Get("part").String())

		uname := Filter(js.Get("uname").String())
		uname = null2Str(uname, Filter(js.Get("title").String()))

		status := Filter(js.Get("status").String())
		status = null2Str(status, Filter(js.Get("page_data").Get("download_title").String()))

		itemId, e := js.Get("itemId").Int()
		if itemId == 0 || e != nil {
			itemId, _ = js.Get("owner_id").Int()
		}
		c.ItemId = strconv.Itoa(itemId)

		// 设置 groupId 和 uid
		c.GroupId = Filter(js.Get("groupId").String())
		c.Uid = Filter(js.Get("uid").String())

		if status != "completed" && status != "视频已缓存完成" && status != "" {
			res.SkippedDirs = append(res.SkippedDirs, v)
			logrus.Warn("未缓存完成,跳过合成", v, title+"-"+uname)
			continue
		}
		if !utils.IsExist(c.OutputDir) {
			_ = os.MkdirAll(c.OutputDir, os.ModePerm)
		}
		groupPath := groupTitle + "-" + uname
		groupDir := filepath.Join(c.OutputDir, groupPath)
		if !utils.IsExist(groupDir) {
			if err = os.MkdirAll(groupDir, os.ModePerm); err != nil {
				return nil, fmt.Errorf("无法创建目录：%s", groupDir)
			}
		}
		// 生成输出文件名
		// mp4Name := title + "-" + part + conver.Mp4Suffix
		if part == "" {
			part = title
		}
		mp4Name := part + conver.Mp4Suffix
		outputFile := filepath.Join(groupDir, mp4Name)

		// 设置元数据信息
		c.Title = title
		c.Uname = uname
		c.GroupTitle = groupTitle

		// 检查是否已经存在已合并文件（选择覆盖时不跳过）
		if !c.Overlay && utils.IsExist(outputFile) {
			// 提取已合并文件的元数据
			metadata, getErr := c.getMp4Metadata(outputFile)
			if getErr == nil {
				// 验证三个值是否一致
				if metadata["title"] == c.GroupId && metadata["artist"] == c.Uid && metadata["album"] == c.ItemId {
					logrus.Warn("跳过已合并文件: ", outputFile)
					continue
				}
			}
			// 如果元数据提取失败或验证失败，仍然跳过同名文件
			logrus.Warn("跳过已合并的视频: ", outputFile)
			continue
		}

		// 检查目录中是否存在与输入音频和视频文件内容相同的文件
		if exists, existingFile := c.isIdenticalFileExists(groupDir, video, audio, part); exists && !c.Overlay {
			logrus.Warn("跳过完全相同的视频: ", existingFile)
			continue
		}

		// 执行合成
		if er := c.Composition(video, audio, outputFile); er != nil {
			logrus.Errorf("%s 合成失败", filepath.Base(outputFile))
			res.FailedFiles = append(res.FailedFiles, outputFile)
			continue
		}

		// 生成并存储文件哈希值，用于后续的重复检测
		hashFile := strings.ReplaceAll(outputFile, ".mp4", ".hash")
		inputHash := c.calculateCombinedHash(video, audio)
		if inputHash != "" {
			_ = os.WriteFile(hashFile, []byte(inputHash), 0644)
		}

		res.OutputFiles = append(res.OutputFiles, filepath.Join(groupPath, mp4Name))
	}
	c.progress(StageMerge, len(dirs), len(dirs), "")

	// 处理未合并的MP3和视频文件
	if c.Summarize && !res.Canceled {
		// 查找未合并的MP3和视频文件
		for _, v := range dirs {
			video, audio, e := c.GetAudioAndVideo(v)
			if e != nil {
				continue
			}

			// 检查是否已经合成
			info := filepath.Join(v, conver.VideoInfoJson)
			if !utils.IsExist(info) {
				info = filepath.Join(v, conver.VideoInfoSuffix)
				if !utils.IsExist(info) {
					info = filepath.Join(v, conver.PlayEntryJson)
					if !utils.IsExist(info) {
						continue
					}
				}
			}
			infoStr, e := os.ReadFile(info)
			if e != nil {
				continue
			}
			js, e := simplejson.NewJson(infoStr)
			if e != nil {
				continue
			}

			groupTitle := Filter(js.Get("groupTitle").String())
			groupTitle = null2Str(groupTitle, Filter(js.Get("owner_name").String()))
			uname := Filter(js.Get("uname").String())
			uname = null2Str(uname, Filter(js.Get("title").String()))
			title := Filter(js.Get("page_data").Get("download_subtitle").String())
			title = null2Str(title, Filter(js.Get("title").String()))

			// 添加空值检查，避免创建空目录名或尝试复制空文件路径
			if groupTitle == "" && uname == "" {
				logrus.Warn("项目信息为空，跳过处理未合并文件: ", v)
				continue
			}

			// 创建项目特定的未合并文件夹
			groupPath := groupTitle + "-" + uname
			groupDir := filepath.Join(c.OutputDir, groupPath)
			if !utils.IsExist(groupDir) {
				if err = os.MkdirAll(groupDir, os.ModePerm); err != nil {
					logrus.Error("创建项目目录失败: ", err)
					continue
				}
			}
			summaryDir := filepath.Join(groupDir, "未合并文件")
			if !utils.IsExist(summaryDir) {
				if err = os.MkdirAll(summaryDir, os.ModePerm); err != nil {
					logrus.Error("创建未合并文件目录失败: ", err)
					continue
				}
			}

			// 复制未合并的视频文件
			if utils.IsExist(video) && title != "" {
				videoDest := filepath.Join(summaryDir, title+"_video"+filepath.Ext(video))
				if !utils.IsExist(videoDest) {
					if err := c.copyFile(video, videoDest); err == nil {
						logrus.Info("已将未合并的视频文件放入汇总目录: ", videoDest)
					}
				} else {
					logrus.Warn("未合并的视频文件已存在，跳过复制: ", videoDest)
				}
			}

			// 复制未合并的音频文件
			if utils.IsExist(audio) && title != "" {
				audioDest := filepath.Join(summaryDir, title+"_audio"+filepath.Ext(audio))
				if !utils.IsExist(audioDest) {
					if err := c.copyFile(audio, audioDest); err == nil {
						logrus.Info("已将未合并的音频文件放入汇总目录: ", audioDest)
					}
				} else {
					logrus.Warn("未合并的音频文件已存在，跳过复制: ", audioDest)
				}
			}
		}
	}

	res.ElapsedSecs = time.Now().Unix() - begin
	return res, nil
}

func (c *Config) findMp4Info(fp, sub string) bool {
	if !utils.IsExist(c.GPACPath) {
		return false
	}
	ret, err := hideWindow(exec.Command(c.GPACPath, "-info", fp)).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(ret), sub)
}

func null2Str(s string, value string) string {
	if s != "" {
		return s
	}
	return value
}

func (c *Config) wait() {
	fmt.Println("按任意键退出程序")
	_, _ = fmt.Scanln()
	os.Exit(0)
}
