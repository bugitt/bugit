package db

import log "unknwon.dev/clog/v2"

func (ptask *PipeTask) CI001() error {

	// load repo
	if err := ptask.LoadRepo(); err != nil {
		return err
	} else {
		log.Info("load repo success for CI task: %d", ptask.ID)
	}
	return nil
}
