package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"miniclaw/internal/models"

	"github.com/robfig/cron/v3"
)

type RunFunc func(ctx context.Context, input models.AgentInput) (models.AgentOutput, error)

type Scheduler struct {
	config  Config
	runFunc RunFunc
}

func NewScheduler(cfg Config, runFunc RunFunc) *Scheduler {
	return &Scheduler{
		config:  cfg,
		runFunc: runFunc,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.config.SchedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.executeDueTasks(ctx)
		}
	}
}

func (s *Scheduler) executeDueTasks(ctx context.Context) {
	tasks, err := s.loadTasks()
	if err != nil {
		log.Printf("error loading tasks: %v", err)
		return
	}

	now := time.Now()
	for _, task := range tasks {
		if task.Status != "active" || task.NextRun == nil {
			continue
		}

		if task.Expires != nil {
			expires, err := time.Parse(time.RFC3339, *task.Expires)
			if err != nil {
				log.Printf("error parsing expires for task %s: %v", task.Filename, err)
			} else if now.After(expires) {
				log.Printf("[task] expired %s (chat=%d type=%s/%s expires=%s prompt=%q), deleting", task.Filename, task.ChatID, task.ScheduleType, task.ScheduleValue, *task.Expires, task.Prompt)
				s.deleteTask(task)
				continue
			}
		}

		nextRun, err := time.Parse(time.RFC3339, *task.NextRun)
		if err != nil {
			log.Printf("error parsing next_run for task %s: %v", task.Filename, err)
			continue
		}

		if now.Before(nextRun) {
			continue
		}

		log.Printf("[task] executing %s (chat=%d schedule=%s/%s)", task.Filename, task.ChatID, task.ScheduleType, task.ScheduleValue)

		_, err = s.runFunc(ctx, models.AgentInput{
			ChatID:          task.ChatID,
			ThreadID:        task.ThreadID,
			Prompt:          task.Prompt,
			IsolatedSession: task.IsolatedSession,
			TaskName:        task.Filename,
		})

		if err != nil {
			log.Printf("[task] error running %s: %v", task.Filename, err)
		}

		newNextRun := s.calculateNextRun(task)
		if newNextRun == nil {
			log.Printf("[task] completed %s (chat=%d prompt=%q), deleting", task.Filename, task.ChatID, task.Prompt)
			s.deleteTask(task)
		} else {
			log.Printf("[task] rescheduled %s next_run=%s", task.Filename, *newNextRun)
			task.NextRun = newNextRun
			s.saveTask(task)
		}
	}
}

func (s *Scheduler) loadTasks() ([]models.Task, error) {
	tasksDir := filepath.Join(s.config.DataDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("reading tasks directory: %w", err)
	}

	var tasks []models.Task
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(tasksDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("error reading task file %s: %v", entry.Name(), err)
			continue
		}

		var task models.Task
		if err := json.Unmarshal(data, &task); err != nil {
			log.Printf("error parsing task file %s: %v", entry.Name(), err)
			continue
		}

		task.Filename = entry.Name()
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *Scheduler) calculateNextRun(task models.Task) *string {
	switch task.ScheduleType {
	case "once":
		return nil

	case "cron":
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, err := parser.Parse(task.ScheduleValue)
		if err != nil {
			log.Printf("error parsing cron expression %q for task %s: %v", task.ScheduleValue, task.Filename, err)
			return nil
		}
		next := sched.Next(time.Now()).Format(time.RFC3339)
		return &next

	case "interval":
		dur, err := time.ParseDuration(task.ScheduleValue)
		if err != nil {
			log.Printf("error parsing interval %q for task %s: %v", task.ScheduleValue, task.Filename, err)
			return nil
		}
		next := time.Now().Add(dur).Format(time.RFC3339)
		return &next

	default:
		log.Printf("unknown schedule type %q for task %s", task.ScheduleType, task.Filename)
		return nil
	}
}

func (s *Scheduler) deleteTask(task models.Task) {
	path := filepath.Join(s.config.DataDir, "tasks", task.Filename)
	if err := os.Remove(path); err != nil {
		log.Printf("[task] error deleting %s: %v", task.Filename, err)
	} else {
		log.Printf("[task] deleted %s", task.Filename)
	}
}

func (s *Scheduler) saveTask(task models.Task) {
	path := filepath.Join(s.config.DataDir, "tasks", task.Filename)
	data, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		log.Printf("error marshaling task %s: %v", task.Filename, err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[task] error saving %s: %v", task.Filename, err)
	} else {
		log.Printf("[task] saved %s (status=%s next_run=%v)", task.Filename, task.Status, task.NextRun)
	}
}
