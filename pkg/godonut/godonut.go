package godonut

import (
	"Rshell/pkg/godonut/gonut"
)

func GenShellcode(fileContent []byte, runParams string, architecture string) ([]byte, error) {
	c := gonut.DefaultConfig()
	c.ModuleName = ""
	c.Server = ""
	c.Entropy = gonut.DONUT_ENTROPY_DEFAULT
	switch architecture {
	case "x64", "amd64":
		c.Arch = gonut.DONUT_ARCH_X64
	case "x86", "386":
		c.Arch = gonut.DONUT_ARCH_X86
	}
	c.Output = ""
	c.Format = gonut.DONUT_FORMAT_BINARY
	c.OEP = 0
	c.ExitOpt = gonut.DONUT_OPT_EXIT_THREAD
	c.Class = ""
	c.Domain = ""
	//c.Input = filePath
	c.InputByte = fileContent
	c.Method = ""
	c.Args = runParams
	c.Unicode = false
	c.Runtime = ""
	c.Thread = false
	c.GonutCompress = gonut.GONUT_COMPRESS_NONE
	c.Bypass = gonut.DONUT_BYPASS_CONTINUE
	c.Headers = gonut.DONUT_HEADERS_OVERWRITE
	c.Decoy = ""
	c.Verbose = false
	o := gonut.New(c)
	if err := o.ValidateLoaderConfig(); err != nil {
		return []byte{}, err
	}

	// 2. get information about the file to execute in memory
	if err := o.ReadFileInfo(); err != nil {
		return []byte{}, err
	}

	// 3. validate the module configuration
	if err := o.ValidateFileInfo(); err != nil {
		return []byte{}, err
	}

	// 4. build the module
	if err := o.BuildModule(); err != nil {
		return []byte{}, err
	}

	// 5. build the instance
	if err := o.BuildInstance(); err != nil {
		return []byte{}, err
	}

	// 6. build the loader
	if err := o.BuildLoader(); err != nil {
		return []byte{}, err
	}
	return o.PicData, nil

}
