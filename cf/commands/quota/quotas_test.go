package quota_test

import (
	"github.com/cloudfoundry/cli/cf/api/quotas/quotasfakes"
	"github.com/cloudfoundry/cli/cf/commands/quota"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/flags"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("quotas command", func() {
	var (
		ui                  *testterm.FakeUI
		config              coreconfig.Repository
		quotaRepo           *quotasfakes.FakeQuotaRepository
		requirementsFactory *testreq.FakeReqFactory
		deps                commandregistry.Dependency
	)

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		deps.Config = config
		deps.RepoLocator = deps.RepoLocator.SetQuotaRepository(quotaRepo)
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("quotas").SetDependency(deps, pluginCall))
	}

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		quotaRepo = new(quotasfakes.FakeQuotaRepository)
		requirementsFactory = &testreq.FakeReqFactory{LoginSuccess: true}
		config = testconfig.NewRepositoryWithDefaults()
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCLICommand("quotas", args, requirementsFactory, updateCommandDependency, false, ui)
	}

	Describe("requirements", func() {
		It("requires the user to be logged in", func() {
			requirementsFactory.LoginSuccess = false
			Expect(runCommand()).ToNot(HavePassedRequirements())
		})

		Context("when arguments are provided", func() {
			var cmd commandregistry.Command
			var flagContext flags.FlagContext

			BeforeEach(func() {
				cmd = &quota.ListQuotas{}
				cmd.SetDependency(deps, false)
				flagContext = flags.NewFlagContext(cmd.MetaData().Flags)
			})

			It("should fail with usage", func() {
				flagContext.Parse("blahblah")

				reqs := cmd.Requirements(requirementsFactory, flagContext)

				err := testcmd.RunRequirements(reqs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Incorrect Usage"))
				Expect(err.Error()).To(ContainSubstring("No argument required"))
			})
		})
	})

	Context("when quotas exist", func() {
		BeforeEach(func() {
			quotaRepo.FindAllReturns([]models.QuotaFields{
				{
					Name:                    "quota-name",
					MemoryLimit:             1024,
					InstanceMemoryLimit:     512,
					RoutesLimit:             111,
					ServicesLimit:           222,
					NonBasicServicesAllowed: true,
					AppInstanceLimit:        -1,
					ReservedRoutePorts:      "4",
				},
				{
					Name:                    "quota-non-basic-not-allowed",
					MemoryLimit:             434,
					InstanceMemoryLimit:     -1,
					RoutesLimit:             1,
					ServicesLimit:           2,
					NonBasicServicesAllowed: false,
					AppInstanceLimit:        10,
					ReservedRoutePorts:      "4",
				},
				{
					Name:                    "quota-unlimited-routes",
					MemoryLimit:             434,
					InstanceMemoryLimit:     1,
					RoutesLimit:             -1,
					ServicesLimit:           2,
					NonBasicServicesAllowed: false,
					AppInstanceLimit:        10,
					ReservedRoutePorts:      "4",
				},
			}, nil)
		})

		It("lists quotas", func() {
			Expect(Expect(runCommand()).To(HavePassedRequirements())).To(HavePassedRequirements())
			Expect(terminal.Decolorize(ui.Outputs[0])).To(Equal("Getting quotas as my-user..."))
			Expect(terminal.Decolorize(ui.Outputs[1])).To(Equal("OK"))
			Expect(terminal.Decolorize(ui.Outputs[3])).To(MatchRegexp("name\\s*total memory\\s*instance memory\\s*routes\\s*service instances\\s*paid plans\\s*app instances\\s*route ports\\s*"))
			Expect(terminal.Decolorize(ui.Outputs[4])).To(MatchRegexp("quota-name\\s*1G\\s*512M\\s*111\\s*222\\s*allowed\\s*unlimited\\s*4"))
			Expect(terminal.Decolorize(ui.Outputs[5])).To(MatchRegexp("quota-non-basic-not-allowed\\s*434M\\s*unlimited\\s*1\\s*2\\s*disallowed\\s*10\\s*4"))
			Expect(terminal.Decolorize(ui.Outputs[6])).To(MatchRegexp("quota-unlimited-routes\\s*434M\\s*1M\\s*unlimited\\s*2\\s*disallowed\\s*10\\s*4"))
		})

		It("displays unlimited services properly", func() {
			quotaRepo.FindAllReturns([]models.QuotaFields{
				{
					Name:                    "quota-with-no-limit-to-services",
					MemoryLimit:             434,
					InstanceMemoryLimit:     1,
					RoutesLimit:             2,
					ServicesLimit:           -1,
					NonBasicServicesAllowed: false,
					AppInstanceLimit:        7,
				},
			}, nil)
			Expect(Expect(runCommand()).To(HavePassedRequirements())).To(HavePassedRequirements())
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"quota-with-no-limit-to-services", "434M", "1M", "2", "unlimited", "disallowed", "7"},
			))

			quotaRepo.FindAllReturns([]models.QuotaFields{
				{
					Name:                    "quota-with-no-limit-to-app-instance",
					MemoryLimit:             434,
					InstanceMemoryLimit:     1,
					RoutesLimit:             2,
					ServicesLimit:           7,
					NonBasicServicesAllowed: false,
					AppInstanceLimit:        -1,
				},
			}, nil)
			Expect(Expect(runCommand()).To(HavePassedRequirements())).To(HavePassedRequirements())
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"quota-with-no-limit-to-app-instance", "434M", "1M", "2", "7", "disallowed", "unlimited"},
			))

			quotaRepo.FindAllReturns([]models.QuotaFields{
				{
					Name:                    "quota-with-no-limit-to-reserved-route-ports",
					MemoryLimit:             434,
					InstanceMemoryLimit:     1,
					RoutesLimit:             2,
					ServicesLimit:           7,
					NonBasicServicesAllowed: false,
					AppInstanceLimit:        7,
					ReservedRoutePorts:      "-1",
				},
			}, nil)
			Expect(Expect(runCommand()).To(HavePassedRequirements())).To(HavePassedRequirements())
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"quota-with-no-limit-to-app-instance", "434M", "1M", "2", "7", "disallowed", "7", "unlimited"},
			))
		})
	})

	Context("when an error occurs fetching quotas", func() {
		BeforeEach(func() {
			quotaRepo.FindAllReturns([]models.QuotaFields{}, errors.New("I haz a borken!"))
		})

		It("prints an error", func() {
			runCommand()
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Getting quotas as", "my-user"},
				[]string{"FAILED"},
			))
		})
	})

})
