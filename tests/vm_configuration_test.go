/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Configurations", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("New VM with different cpu topologies give", func() {

		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = tests.NewRandomVMWithEphemeralDisk("kubevirt/alpine-registry-disk-demo:devel")
		})
		It("should report 3 cpu cores", func() {
			vm.Spec.Domain.CPU = &v1.CPU{
				Cores: 3,
			}

			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to Alpine"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login"},
				&expect.BSnd{S: "root\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
				&expect.BExp{R: "3"},
			}, 250*time.Second)

			Expect(err).ToNot(HaveOccurred())
		}, 300)
	})

	Context("New VM with explicitly set VirtIO drives", func() {

		var vm *v1.VirtualMachine
		var diskDev v1.DiskDevice

		BeforeEach(func() {
			diskDev = v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: "virtio",
				},
			}
			vm = tests.NewRandomVMWithDirectLunAndDevice(2, false, diskDev)
		})
		It("should have /dev/vda node", func() {
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to Alpine"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login"},
				&expect.BSnd{S: "root\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "ls /dev/vda\n"},
				&expect.BExp{R: "/dev/vda"},
			}, 150*time.Second)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("New VM with explicitly set SATA drives", func() {

		var vm *v1.VirtualMachine
		var diskDev v1.DiskDevice

		BeforeEach(func() {
			diskDev = v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: "sata",
				},
			}
			vm = tests.NewRandomVMWithDirectLunAndDevice(2, false, diskDev)
		})
		It("should have /dev/sda node", func() {
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to Alpine"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login"},
				&expect.BSnd{S: "root\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "ls /dev/sda\n"},
				&expect.BExp{R: "/dev/sda"},
			}, 150*time.Second)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("New VM with all supported drives", func() {

		var vm *v1.VirtualMachine

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := "kubevirt/cirros-registry-disk-demo:devel"
			// virtio - added by NewRandomVMWithEphemeralDisk
			vm = tests.NewRandomVMWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vm, "disk2", "sata", containerImage)
			// ide
			tests.AddEphemeralDisk(vm, "disk3", "ide", containerImage)
			// floppy
			tests.AddEphemeralFloppy(vm, "disk4", containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda, hda, fda

			// We need ide support for the test, q35 does not support ide
			vm.Spec.Domain.Machine.Type = "pc"
		})

		// FIXME ide and floppy is not recognized by the used image right now
		It("should have all the device nodes", func() {
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "login as 'cirros' user. default password: 'gocubsgo'. use 'sudo' for root."},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "cirros login:"},
				&expect.BSnd{S: "cirros\n"},
				&expect.BExp{R: "Password:"},
				&expect.BSnd{S: "gocubsgo\n"},
				&expect.BExp{R: "$"},
				// keep the ordering!
				&expect.BSnd{S: "ls /dev/sda  /dev/vda  /dev/vdb\n"},
				&expect.BExp{R: "/dev/sda  /dev/vda  /dev/vdb"},
			}, 150*time.Second)

			Expect(err).ToNot(HaveOccurred())
		})
	})

})
