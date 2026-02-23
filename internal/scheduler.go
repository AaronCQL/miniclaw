package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"goclaw/internal/models"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	config      Config
	agentRunner *AgentRunner
	bot         *TelegramBot
}

func NewScheduler(cfg Config, agentRunner *AgentRunner, bot *TelegramBot) *Scheduler {
	return &Scheduler{
		config:      cfg,
		agentRunner: agentRunner,
		bot:         bot,
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

		nextRun, err := time.Parse(time.RFC3339, *task.NextRun)
		if err != nil {
			log.Printf("error parsing next_run for task %s: %v", task.Filename, err)
			continue
		}

		if now.Before(nextRun) {
			continue
		}

		log.Printf("[task] executing %s (chat=%d schedule=%s/%s)", task.Filename, task.ChatID, task.ScheduleType, task.ScheduleValue)

		output, err := s.agentRunner.Run(ctx, models.AgentInput{
			ChatID: task.ChatID,
			Prompt: task.Prompt,
		})

		if err == nil && output.Result != "" {
			s.bot.SendMessage(task.ChatID, output.Result)
		} else if err != nil {
			log.Printf("[task] error running %s: %v", task.Filename, err)
		}

		// Update next_run or mark as completed
		newNextRun := s.calculateNextRun(task)
		if newNextRun == nil {
			task.Status = "completed"
			log.Printf("[task] completed %s (no next run)", task.Filename)
		} else {
			log.Printf("[task] rescheduled %s next_run=%s", task.Filename, *newNextRun)
		}
		task.NextRun = newNextRun
		s.saveTask(task)
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

func (s *Scheduler) saveTask(task models.Task) {
	path := filepath.Join(s.config.DataDir, "tasks", task.Filename)
	data, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		log.Printf("error marshaling task %s: %v", task.Filename, err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("error writing task file %s: %v", task.Filename, err)
	}
}
