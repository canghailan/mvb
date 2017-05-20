package main

import (
	"./mvb"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
	"sync"
)

var (
	app     = kingpin.New(os.Args[0], "多版本备份工具")
	verbose = app.Flag("verbose", "输出调试信息").Short('v').Bool()

	initCommand = app.Command("init", "初始化当前文件夹作为备份存储空间")
	initPath    = initCommand.Arg("path", "要备份的文件夹").Required().String()

	previewCommand = app.Command("preview", "预览将要备份的版本")

	backupCommand = app.Command("backup", "备份")

	checkCommand = app.Command("check", "检查最新存储的文件是否完整（与备份文件夹中的当前文件对比）")

	restoreCommand = app.Command("restore", "还原")
	restoreVersion = restoreCommand.Arg("version", "要还原的版本，默认为最新版本").Default("").String()
	restorePath    = restoreCommand.Arg("path", "要还原到的文件夹，默认为备份文件夹").Default("").String()

	linkCommand = app.Command("link", "通过符号链接，创建版本文件视图")
	linkVersion = linkCommand.Arg("version", "要链接的版本").Required().String()
	linkPath    = linkCommand.Arg("path", "要链接的文件夹，必须存在且为空文件夹").Required().String()

	listCommand = app.Command("list", "查看所有备份版本")
	listVersion = listCommand.Arg("version", "短版本").Default("").String()

	getCommand = app.Command("get", "读取备份内容")
	getVersion = getCommand.Arg("version", "版本与路径同时为空时，读取版本反向索引；版本不为空时，读取版本特定数据").Default("").String()
	getPath    = getCommand.Arg("path", "路径为空时，读取版本快照；路径不为空时，读取该版本文件内容").Default("").String()

	diffCommand  = app.Command("diff", "对比两个版本的差异")
	diffVersionA = diffCommand.Arg("version a", "版本A，默认为最新版本").Default("").String()
	diffVersionB = diffCommand.Arg("version b", "版本B，默认为将要备份的版本").Default("").String()

	deleteCommand = app.Command("delete", "删除指定的版本")
	deleteVersion = deleteCommand.Arg("version", "版本").Required().String()

	gcCommand = app.Command("gc", "清理备份存储空间，删除无用文件")
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))
	mvb.Verbose = *verbose
	switch command {
	case initCommand.FullCommand():
		executeInitCommand()
	case backupCommand.FullCommand():
		executeBackupCommand()
	case restoreCommand.FullCommand():
		executeRestoreCommand()
	case linkCommand.FullCommand():
		executeLinkCommand()
	case listCommand.FullCommand():
		executeListCommand()
	case getCommand.FullCommand():
		executeGetCommand()
	case deleteCommand.FullCommand():
		executeDeleteCommand()
	case diffCommand.FullCommand():
		executeDiffCommand()
	case previewCommand.FullCommand():
		executePreviewCommand()
	case checkCommand.FullCommand():
		executeCheckCommand()
	case gcCommand.FullCommand():
		executeGcCommand()
	}
}

func executeInitCommand() {
	path := *initPath

	if err := ioutil.WriteFile("ref", []byte(path), 0644); err != nil {
		mvb.Errorf("%v", err)
	}
	mvb.Verbosef("初始化路径: %s", path)
}

func executeBackupCommand() {
	timestamp := time.Now()
	files := mvb.GetRefFiles()
	version := mvb.StringifyVersionObject(files)
	versionSha1 := mvb.Sha1([]byte(version))

	if !mvb.IsObjectExist(versionSha1) {
		mvb.CopyObjects(files)
		mvb.WriteVersionObject(versionSha1, version)
		mvb.AddVersionToIndex(mvb.Version{Sha1: versionSha1, Timestamp: timestamp.Format(mvb.ISO8601)})
	} else {
		mvb.Verbosef("版本已存在： %s\n", versionSha1)
	}

	println(versionSha1)
}

func executeRestoreCommand() {
	version := *restoreVersion
	root := *restorePath

	if version == "" {
		version = mvb.GetLatestVersionSha1()
		if version == "" {
			mvb.Errorf("版本不存在：%s", *restoreVersion)
		}
	} else {
		version = mvb.ResolveVersionSha1(version)
	}
	if root == "" {
		root = mvb.GetRef()
	}

	src := mvb.GetFiles(root)
	dst := mvb.GetVersionFiles(version)

	mvb.FastGetFilesSha1(src, dst)
	mvb.GetFilesSha1(root, src)

	diffFiles := mvb.DiffFiles(src, dst)
	for i := len(diffFiles) - 1; i>=0;i-- {
		f := diffFiles[i]
		p := filepath.Join(root, f.Path)

		mvb.Verbosef("%s %s\n", f.Type, f.Path)
		if f.Type == "+" || f.Type == "*" {
			if !strings.HasSuffix(f.Path, "/") {
				mvb.CopyFile(mvb.GetObjectPath(f.Sha1), p)
			}
		} else if f.Type == "-" {
			if err := os.Remove(p); err != nil {
				mvb.Errorf("删除文件失败：%s", p)
			}
		}
	}
}

func executeLinkCommand() {
	version := *linkVersion
	path := *linkPath

	fis, err := ioutil.ReadDir(path)
	if err != nil {
		mvb.Errorf("%v", err)
	}
	if len(fis) > 0 {
		mvb.Errorf("%s 不是空文件夹", path)
	}

	version = mvb.ResolveVersionSha1(version)
	fileObjects := mvb.GetVersionFiles(version)
	for _, f := range fileObjects {
		if strings.HasSuffix(f.Path, "/") {
			if err := os.Mkdir(filepath.Join(path, f.Path), os.ModeDir|0755); err != nil {
				mvb.Errorf("%v", err)
			}
		} else {
			fileObject, err := filepath.Abs(mvb.GetObjectPath(f.Sha1))
			if err != nil {
				mvb.Errorf("%v", err)
			}
			if err := os.Symlink(fileObject, filepath.Join(path, f.Path)); err != nil {
				mvb.Errorf("%v", err)
			}
		}
	}
}

func executeListCommand() {
	pattern := *listVersion

	if pattern == "" {
		mvb.WriteReverseIndexTo(os.Stdout)
	} else if strings.HasPrefix(pattern, "v") {
		r := mvb.GetIndexVersionAt(mvb.ParseIndexedVersion(pattern))
		println(r)
	} else {
		for _, r := range mvb.FindIndexVersions(pattern) {
			println(r)
		}
	}
}

func executeGetCommand() {
	version := *getVersion
	path := *getPath

	if version == "" && path == "" {
		mvb.WriteReverseIndexTo(os.Stdout)
		return
	}

	version = mvb.ResolveVersionSha1(version)
	if path == "" {
		mvb.WriteObjectTo(version, os.Stdout)
		return
	}

	files := mvb.GetVersionFiles(version)

	if strings.HasSuffix(path, "/") {
		for _, f := range files {
			if strings.HasPrefix(f.Path, path) && f.Path != path {
				print(mvb.StringifyFileMetadata(f))
			}
		}
		return
	}

	file := mvb.SearchFile(files, path)
	if file == nil {
		mvb.Errorf("文件不存在：%s %s", version, path)
	}
	mvb.WriteObjectTo(file.Sha1, os.Stdout)
}

func executeDeleteCommand() {
	pattern := *deleteVersion

	if strings.HasPrefix(pattern, "v") {
		mvb.DeleteIndexVersionAt(mvb.ParseIndexedVersion(pattern))
	} else {
		mvb.DeleteIndexVersion(pattern)
	}
}

func executeDiffCommand() {
	versionA := *diffVersionA
	versionB := *diffVersionB

	if versionA == "" {
		versionA = mvb.GetLatestVersionSha1()
		if versionA == "" {
			mvb.Errorf("版本A不存在：%s", *diffVersionA)
		}
	} else {
		versionA = mvb.ResolveVersionSha1(versionA)
	}
	filesA := mvb.GetVersionFiles(versionA)

	var filesB []mvb.FileMetadata
	if versionB == "" {
		root := mvb.GetRef()
		filesB = mvb.GetFiles(root)
		mvb.GetFilesSha1(root, filesB)
	} else {
		versionB = mvb.ResolveVersionSha1(versionB)
		filesB = mvb.GetVersionFiles(versionB)
	}

	diffFileObjects := mvb.DiffFiles(filesA, filesB)
	for _, f := range diffFileObjects {
		fmt.Printf("%s %s\n", f.Type, f.Path)
	}
}

func executePreviewCommand() {
	files := mvb.GetRefFiles()
	version := mvb.StringifyVersionObject(files)
	versionSha1 := mvb.Sha1([]byte(version))

	println(version)
	println(versionSha1)
}

func executeCheckCommand() {
	var wg sync.WaitGroup
	sem := make(chan int, mvb.MAX_GOS)
	filepath.Walk("objects", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			mvb.Errorf("%v", err)
		}
		if fi.IsDir() {
			return nil
		}
		p, err := filepath.Rel("objects", path)
		if err != nil {
			mvb.Errorf("%v", err)
		}

		sem <- 1
		wg.Add(1)
		go func() {
			s1 := p[:2] + p[3:]
			s2 := mvb.GetFileSha1(path)
			mvb.Verbosef("检查：%s\n", path)
			if s1 != s2 {
				println(path)
			}
			wg.Done()
			<-sem
		}()
		return nil
	})
	wg.Wait()
}

func executeGcCommand() {
	objects := map[string]bool{}

	for _, v := range mvb.GetIndexVersions() {
		s := mvb.ParseVersion(v).Sha1
		objects[s] = true
		for _, f := range mvb.GetVersionFiles(s) {
			objects[f.Sha1] = true
		}
	}

	filepath.Walk("objects", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			mvb.Errorf("%v", err)
		}
		if fi.IsDir() {
			return nil
		}
		p, err := filepath.Rel("objects", path)
		if err != nil {
			mvb.Errorf("%v", err)
		}


		s := p[:2] + p[3:]
		if _, ok := objects[s]; !ok {
			println(s)
			mvb.Verbosef("删除：%s\n", path)
			if err := os.Remove(path); err != nil {
				mvb.Errorf("%v", err)
			}
		} else {
			mvb.Verbosef("保留：%s\n", path)
		}
		return nil
	})

	ls, err := ioutil.ReadDir("objects")
	if err != nil {
		mvb.Errorf("%v", err)
	}
	for _, f := range ls {
		if f.IsDir() {
			p := filepath.Join("objects", f.Name())
			c, err := ioutil.ReadDir(p)
			if err != nil {
				mvb.Errorf("%v", err)
			}
			if len(c) == 0 {
				mvb.Verbosef("删除：%s\n", p)
				if err := os.Remove(p); err != nil {
					mvb.Errorf("%v", err)
				}
			}
		}
	}
}