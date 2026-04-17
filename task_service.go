package main

import taskservice "flux-board/internal/service/task"

type TaskService = taskservice.Service

func (a *App) taskService() TaskService {
	if a.taskSvc != nil {
		return a.taskSvc
	}
	return taskservice.New(a.taskRepository())
}
