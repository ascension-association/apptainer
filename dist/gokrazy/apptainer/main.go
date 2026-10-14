// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Command apptainer is a gokrazy package wrapping a statically linked
// apptainer installation shipped as extra files under /usr/local/apptainer.
//
// gokrazy's root file system is read-only, so this wrapper points apptainer's
// home, cache and temporary directories at the writable /perm partition
// before executing the real apptainer binary.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	apptainerBin = "/usr/local/apptainer/bin/apptainer"
	libexecBin   = "/usr/local/apptainer/libexec/apptainer/bin"
)

// stateDir returns a writable directory to hold apptainer state, preferring
// the persistent /perm partition and falling back to the /tmp tmpfs.
func stateDir() string {
	for _, dir := range []string{"/perm/apptainer", "/tmp/apptainer"} {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		}
	}
	return ""
}

// setDefault sets the environment variable key to a directory below base
// unless it is already set, creating the directory as needed.
func setDefault(key, base, name string) {
	if os.Getenv(key) != "" {
		return
	}
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "apptainer-gokrazy: creating %s: %v\n", dir, err)
		return
	}
	os.Setenv(key, dir)
}

func main() {
	// gokrazy starts every package on boot. There is nothing to supervise,
	// so tell gokrazy not to restart us (same as gokrazy.DontStartOnBoot).
	if os.Getenv("GOKRAZY_FIRST_START") == "1" {
		os.Exit(125)
	}

	if base := stateDir(); base != "" {
		if home := os.Getenv("HOME"); home == "" || home == "/" {
			os.Unsetenv("HOME")
			setDefault("HOME", base, "home")
		}
		setDefault("APPTAINER_CACHEDIR", base, "cache")
		setDefault("APPTAINER_TMPDIR", base, "tmp")

		// The bundled apptainer.conf is on the read-only root file system,
		// allow overriding it with a copy on the writable partition.
		conf := filepath.Join(base, "apptainer.conf")
		if _, err := os.Stat(conf); err == nil && os.Getenv("APPTAINER_CONFIG_FILE") == "" {
			os.Setenv("APPTAINER_CONFIG_FILE", conf)
		}
	}

	// apptainer bind mounts /var/tmp into containers (mount tmp = yes).
	// On gokrazy /var is a symlink to /perm/var, which lacks a tmp directory.
	if _, err := os.Stat("/var/tmp"); os.IsNotExist(err) {
		if err := os.MkdirAll("/var/tmp", 0o777); err == nil {
			os.Chmod("/var/tmp", 0o1777)
		}
	}

	// gokrazy has no /usr/bin, /bin etc. Make the bundled helper programs
	// (mksquashfs, unsquashfs) available to apptainer.
	path := libexecBin
	if p := os.Getenv("PATH"); p != "" {
		path += ":" + p
	}
	os.Setenv("PATH", path)

	argv := append([]string{"apptainer"}, os.Args[1:]...)
	if err := syscall.Exec(apptainerBin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "apptainer-gokrazy: executing %s: %v\n", apptainerBin, err)
		fmt.Fprintln(os.Stderr, "apptainer-gokrazy: was dist/gokrazy/build.sh run before building the gokrazy image?")
		os.Exit(1)
	}
}
