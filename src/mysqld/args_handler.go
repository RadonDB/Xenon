/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysqld

// ArgsHandler for mysqld operation.
type ArgsHandler interface {
	// generate mysqld start args
	GenerateStartCmd() []string

	// generate mysqld stop args
	GenerateStopCmd() []string

	// generate mysqld isrunning args
	GenerateIsRunningCmd() []string

	// generate mysqld kill args
	GenerateKillCmd() []string
}
