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
		initialize(*initPath)
	case listCommand.FullCommand():
		list(*listVersion)
	case getCommand.FullCommand():
		get(*getVersion, *getPath)
	case diffCommand.FullCommand():
		diff(*diffVersionA, *diffVersionB)
	case previewCommand.FullCommand():
		preview()
	case backupCommand.FullCommand():
		backup()
	case checkCommand.FullCommand():
		check()
	case restoreCommand.FullCommand():
		restore(*restoreVersion, *restorePath)
	case linkCommand.FullCommand():
		link(*linkVersion, *linkPath)
	case deleteCommand.FullCommand():
		del(*deleteVersion)
	case gcCommand.FullCommand():
		gc()
	}
}

func initialize(path string) {
	if err := ioutil.WriteFile("ref", []byte(path), 0644); err != nil {
		mvb.Errorf("init: %v", err)
	}
	mvb.Verbosef("init: %s", path)
}

// mvb list
func list(pattern string) {
	if pattern == "" {
		i, err := mvb.NewReverseIndex()
		if err != nil {
			mvb.Errorf("读取索引文件错误: %v", err)
		}
		defer i.Close()

		for {
			r := i.NextVersionRecord()
			if r == "" {
				break
			}
			println(r)
		}
	} else if strings.HasPrefix(pattern, "v") {
		r := mvb.FindIndexVersionRecordAt(pattern[1:])
		println(r)
	} else {
		for _, r := range mvb.FindIndexVersionRecords(pattern) {
			println(r)
		}
	}
}

func get(version string, path string) {
	if version == "" && path == "" {
		list("")
		return
	}

	version = mvb.ResolveVersion(version)
	if path == "" {
		mvb.CopyObject(version, os.Stdout)
		return
	}

	fileObjects := mvb.GetVersionFileObjects(version)
	fileObject := mvb.SearchFileObjects(fileObjects, path)
	if fileObject == nil {
		mvb.Errorf("文件不存在：%s %s", version, path)
	}
	mvb.CopyObject(fileObject.DataDigest, os.Stdout)
}

// mvb diff [version a] [version b]
func diff(versionA string, versionB string) {
	if versionA == "" {
		versionA = mvb.GetLatestVersion()
	} else {
		versionA = mvb.ResolveVersion(versionA)
	}
	fileObjectsA := mvb.GetVersionFileObjects(versionA)

	var fileObjectsB []mvb.FileObject
	if versionB == "" {
		root := mvb.GetRef()
		fileObjectsB = mvb.GetFileObjects(root)
		mvb.DigestFileObjects(root, fileObjectsB)
	} else {
		versionB = mvb.ResolveVersion(versionB)
		fileObjectsB = mvb.GetVersionFileObjects(versionB)
	}

	diffFileObjects := mvb.DiffFileObjects(fileObjectsA, fileObjectsB)
	for _, f := range diffFileObjects {
		fmt.Printf("%s %s\n", f.Type, f.Path)
	}
}

func preview() {
	fileObjects := mvb.GetRefFileObjects()
	snapshot := mvb.ToVersionSnapshot(fileObjects)
	id := mvb.Sha1([]byte(snapshot))

	println(snapshot)
	println(id)
}

func backup() {
	timestamp := time.Now()
	fileObjects := mvb.GetRefFileObjects()
	snapshot := mvb.ToVersionSnapshot(fileObjects)
	id := mvb.Sha1([]byte(snapshot))

	if !mvb.IsObjectExist(id) {
		mvb.CopyFileObjects(fileObjects)
		mvb.WriteVersionFile(id, snapshot)
		mvb.AddVersionRecordToIndex(id, timestamp)
	} else {
		mvb.Verbosef("版本已存在： %s\n", id)
	}

	println(id)
}

func check() {
	version := mvb.GetLatestVersion()
	fileObjects := mvb.GetVersionFileObjects(version)
	for _, f := range fileObjects {
		if strings.HasSuffix(f.Path, "/") {
			continue
		}

		src := filepath.Join(mvb.GetRef(), f.Path)
		dst := mvb.GetObjectPath(f.DataDigest)

		s, err := os.Stat(src)
		if err != nil {
			mvb.Errorf("check: %v", err)
		}
		d, err := os.Stat(dst)
		if err != nil {
			mvb.Errorf("check: %v", err)
		}

		if s.Size() != d.Size() {
			fmt.Printf("%s %s\n", dst, f.Path)
		}
	}
}

// mvb restore [version] [path]
func restore(version string, root string) {
	if version == "" {
		version = mvb.GetLatestVersion()
	} else {
		version = mvb.ResolveVersion(version)
	}
	if root == "" {
		root = mvb.GetRef()
	}

	src := mvb.GetFileObjects(root)
	dst := mvb.GetVersionFileObjects(version)

	mvb.FastDigestFileObjects(src, dst)
	mvb.DigestFileObjects(root, src)

	diffFileObjects := mvb.DiffFileObjects(src, dst)
	for i := len(diffFileObjects) - 1; i>=0;i-- {
		f := diffFileObjects[i]
		p := filepath.Join(root, f.Path)

		mvb.Verbosef("%s %s\n", f.Type, f.Path)
		if f.Type == "+" || f.Type == "*" {
			if !strings.HasSuffix(f.Path, "/") {
				mvb.CopyFile(mvb.GetObjectPath(f.DataDigest), p)
			}
		} else if f.Type == "-" {
			if err := os.Remove(p); err != nil {
				mvb.Errorf("删除文件失败：%s", p)
			}
		}
	}
}

// mvb link [version] [path]
func link(version string, path string) {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		mvb.Errorf("link: %v", err)
	}
	if len(fis) > 0 {
		mvb.Errorf("link: %s is not empty dir", path)
	}

	version = mvb.ResolveVersion(version)
	fileObjects := mvb.GetVersionFileObjects(version)
	for _, f := range fileObjects {
		if strings.HasSuffix(f.Path, "/") {
			if err := os.Mkdir(filepath.Join(path, f.Path), os.ModeDir|0755); err != nil {
				mvb.Errorf("link: %v", err)
			}
		} else {
			fileObject, err := filepath.Abs(mvb.GetObjectPath(f.DataDigest))
			if err != nil {
				mvb.Errorf("link: %v", err)
			}
			if err := os.Symlink(fileObject, filepath.Join(path, f.Path)); err != nil {
				mvb.Errorf("link: %v", err)
			}
		}
	}
}

// mvb delete [version]
func del(version string) {
	mvb.Errorf("delete: not supported")
}

// mvb gc
func gc() {
	mvb.Errorf("gc: not supported")
}
