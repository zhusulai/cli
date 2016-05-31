package domain

import (
	"github.com/cloudfoundry/cli/cf/api"
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/errors"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/flags"
)

type DeleteSharedDomain struct {
	ui         terminal.UI
	config     coreconfig.Reader
	orgReq     requirements.TargetedOrgRequirement
	domainRepo api.DomainRepository
}

func init() {
	commandregistry.Register(&DeleteSharedDomain{})
}

func (cmd *DeleteSharedDomain) MetaData() commandregistry.CommandMetadata {
	fs := make(map[string]flags.FlagSet)
	fs["f"] = &flags.BoolFlag{ShortName: "f", Usage: T("Force deletion without confirmation")}

	return commandregistry.CommandMetadata{
		Name:        "delete-shared-domain",
		Description: T("Delete a shared domain"),
		Usage: []string{
			T("CF_NAME delete-shared-domain DOMAIN [-f]"),
		},
		Flags: fs,
	}
}

func (cmd *DeleteSharedDomain) Requirements(requirementsFactory requirements.Factory, fc flags.FlagContext) []requirements.Requirement {
	if len(fc.Args()) != 1 {
		cmd.ui.Failed(T("Incorrect Usage. Requires an argument\n\n") + commandregistry.Commands.CommandUsage("delete-shared-domain"))
	}

	loginReq := requirementsFactory.NewLoginRequirement()
	cmd.orgReq = requirementsFactory.NewTargetedOrgRequirement()

	reqs := []requirements.Requirement{
		loginReq,
		cmd.orgReq,
	}

	return reqs
}

func (cmd *DeleteSharedDomain) SetDependency(deps commandregistry.Dependency, pluginCall bool) commandregistry.Command {
	cmd.ui = deps.UI
	cmd.config = deps.Config
	cmd.domainRepo = deps.RepoLocator.GetDomainRepository()
	return cmd
}

func (cmd *DeleteSharedDomain) Execute(c flags.FlagContext) error {
	domainName := c.Args()[0]
	force := c.Bool("f")

	cmd.ui.Say(T("Deleting domain {{.DomainName}} as {{.Username}}...",
		map[string]interface{}{
			"DomainName": terminal.EntityNameColor(domainName),
			"Username":   terminal.EntityNameColor(cmd.config.Username())}))

	domain, err := cmd.domainRepo.FindByNameInOrg(domainName, cmd.orgReq.GetOrganizationFields().GUID)
	switch err.(type) {
	case nil:
		if !domain.Shared {
			return errors.New(T("domain {{.DomainName}} is an owned domain, not a shared domain.",
				map[string]interface{}{"DomainName": domainName}))
		}
	case *errors.ModelNotFoundError:
		cmd.ui.Ok()
		cmd.ui.Warn(err.Error())
		return nil
	default:
		return errors.New(T("Error finding domain {{.DomainName}}\n{{.Err}}",
			map[string]interface{}{
				"DomainName": domainName,
				"Err":        err.Error()}))
	}

	if !force {
		answer := cmd.ui.Confirm(T("This domain is shared across all orgs.\nDeleting it will remove all associated routes, and will make any app with this domain unreachable.\nAre you sure you want to delete the domain {{.DomainName}}? ", map[string]interface{}{"DomainName": domainName}))

		if !answer {
			return nil
		}
	}

	err = cmd.domainRepo.DeleteSharedDomain(domain.GUID)
	if err != nil {
		return errors.New(T("Error deleting domain {{.DomainName}}\n{{.Err}}",
			map[string]interface{}{"DomainName": domainName, "Err": err.Error()}))
	}

	cmd.ui.Ok()
	return nil
}
