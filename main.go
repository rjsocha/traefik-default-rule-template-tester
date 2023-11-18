package main

import (
	"flag"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
	"time"
)

type Model struct {
	Name          string            `yaml:"name"`
	ContainerName string            `yaml:"containerName"`
	Labels        map[string]string `yaml:"labels"`
}

const threshold = 300 * time.Millisecond

func GetType(v interface{}) string {
	return reflect.TypeOf(v).String()
}

func parseTemplate(filename string, model_filename string) {
	file, err := os.Open(model_filename)
	if err != nil {
		log.Fatalf("Error opening model file: %v", err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading model file: %v", err)
	}

	var model Model
	err = yaml.Unmarshal(content, &model)
	if err != nil {
		log.Fatalf("Error unmarshaling model YAML: %v", err)
	}

	baseName := filepath.Base(filename)

	defaultFuncMap := sprig.TxtFuncMap()
	defaultFuncMap["GetType"] = GetType

	tmpl, err := template.New(baseName).Funcs(defaultFuncMap).ParseFiles(filename)
	if err != nil {
		fmt.Printf("TEMPLATE ERROR: %v\n", err)
		return
	}

	//fmt.Println("Template Definitions:")
	//for _, tpl := range tmpl.Templates() {
	//	fmt.Printf("  - %s\n", tpl.Name())
	//}

	err = tmpl.ExecuteTemplate(os.Stdout, baseName, model)
	if err != nil {
		fmt.Printf("TEMPLATE EXECUTE ERROR: %v\n", err)
	}
	fmt.Printf("\n...\n")
}

func main() {
	file := flag.String("input", "template", "file to watch")
	model := flag.String("model", "model.yaml", "model file")
	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Unable to create new watcher instance: %v", err)
	}
	defer watcher.Close()

	done := make(chan bool)

	abs, _ := filepath.Abs(*file)
	watch := filepath.Base(*file)
	path := filepath.Dir(abs)

	go func() {
		startTime := time.Now()
		for {
			select {
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Fatalf("WATCHER ERROR: %v", err)
			case e, ok := <-watcher.Events:
				if !ok {
					return
				}
				name := filepath.Base(e.Name)
				if name == watch {
					if e.Has(fsnotify.Write) {
						endTime := time.Now()
						duration := endTime.Sub(startTime)
						if duration >= threshold {
							startTime = endTime
							parseTemplate(e.Name, *model)
						}
					}
				}
			}
		}
	}()

	st, err := os.Lstat(path)
	if err != nil {
		log.Fatalf("Unable to locate folder: %v", err)
	}
	if !st.IsDir() {
		log.Fatalf("Unable to locate folder: %v", err)
	}

	fmt.Printf("Write anything to file: %v/%v to trigger template evaluation\n", path, watch)
	err = watcher.Add(path)
	if err != nil {
		log.Fatalf("Unable to watch folder: %v", err)
	}
	<-done
}
