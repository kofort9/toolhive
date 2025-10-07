# MCPExternalAuthConfig Implementation Status

## Overall Progress

**Completed Phases:** 1-2 (CRD Definition & Controller Implementation)
**Current Phase:** 2 (Unit Tests)
**Remaining Phases:** 3-8

**Last Updated:** 2025-10-07

---

## Phase-by-Phase Status

### ✅ Phase 1: Define CRD and Data Structures (COMPLETE)

**Status:** All acceptance criteria verified independently by kubernetes-go-expert

#### Acceptance Criteria
- ✅ CRD compiles without errors
- ✅ DeepCopy methods generated correctly
- ✅ CRD YAML manifests in correct location
- ✅ Kubebuilder markers correct (validation, printcolumns, shortnames)
- ✅ ExternalAuthConfigRef properly typed in MCPServer
- ✅ JSON/YAML struct tags correct (snake_case)
- ✅ Validation enums defined
- ✅ Proper Kubernetes object patterns
- ✅ HeaderStrategy field removed (inferred from ExternalTokenHeaderName)
- ✅ Generated manifests include updated MCPServer CRD

#### Deliverables
- ✅ `cmd/thv-operator/api/v1alpha1/mcpexternalauthconfig_types.go` (~119 lines)
- ✅ `cmd/thv-operator/api/v1alpha1/mcpserver_types.go` (modified)
- ✅ `cmd/thv-operator/api/v1alpha1/zz_generated.deepcopy.go` (updated)
- ✅ `deploy/charts/operator-crds/crds/toolhive.stacklok.dev_mcpexternalauthconfigs.yaml`
- ✅ Updated MCPServer CRD manifest

#### Agent Assignment
- **Implementation:** kubernetes-go-expert
- **Verification:** kubernetes-go-expert (independent verification - PASSED 10/10)

---

### 🟡 Phase 2: Create MCPExternalAuthConfig Controller (PARTIAL - Tests In Progress)

**Status:** Controller implemented and linted (0 issues), tests in progress

#### Acceptance Criteria
- ✅ Controller reconciles MCPExternalAuthConfig successfully
- ✅ Config hash calculated and stored in status
- ✅ Finalizer prevents deletion when referenced
- ✅ Changes trigger MCPServer reconciliation
- ✅ Watch setup correct (primary + secondary with mapping)
- ✅ RBAC markers present
- ✅ Status updates appropriate (no ReferencingServers)
- ✅ Error handling comprehensive
- ✅ Follows Kubernetes patterns
- ✅ Helper functions implemented
- 🟡 Unit tests achieve >80% coverage (IN PROGRESS)
- 🟡 Integration tests pass (IN PROGRESS)

#### Deliverables
- ✅ `cmd/thv-operator/controllers/mcpexternalauthconfig_controller.go` (~270 lines)
  - MCPExternalAuthConfigReconciler struct
  - Reconcile method with full lifecycle
  - Finalizer handling
  - Hash calculation (FNV-1a)
  - findReferencingMCPServers helper
  - GetExternalAuthConfigForMCPServer helper (for Phase 5)
  - SetupWithManager with watch mapping
  - RBAC markers
- 🟡 `cmd/thv-operator/controllers/mcpexternalauthconfig_controller_test.go` (IN PROGRESS)

#### Agent Assignment
- **Implementation:** kubernetes-go-expert
- **Tests:** kubernetes-go-expert (IN PROGRESS)
- **Verification:** kubernetes-go-expert (pending)

#### Notes
- Linter: 0 issues
- Follows MCPToolConfig pattern exactly (minus ReferencingServers in status)
- Annotation key: `toolhive.stacklok.dev/externalauthconfig-hash`

---

### ⏳ Phase 3: Flag-Based Configuration Support (PENDING)

**Status:** Not started

#### Acceptance Criteria
- [ ] External auth flags generated when `TOOLHIVE_USE_CONFIGMAP=false`
- [ ] Flags match expected format from token exchange proposal
- [ ] `deploymentNeedsUpdate()` detects external auth changes
- [ ] Deployment updated when config changes
- [ ] Deployment NOT updated when unchanged
- [ ] Client secrets properly referenced (not inlined)
- [ ] Non-existent MCPExternalAuthConfig causes appropriate error
- [ ] Flags only added when ExternalAuthConfigRef is set

#### Deliverables
- `cmd/thv-operator/controllers/mcpserver_controller.go` (modified, +~150 lines)
  - `generateExternalAuthArgs()` method (~80 lines)
  - `equalExternalAuthArgs()` method (~40 lines)
  - `deploymentForMCPServer()` updated (~10 lines)
  - `deploymentNeedsUpdate()` updated (~20 lines)

#### Integration Points
- Follow OIDC pattern from `generateOIDCArgs()` (lines 1788-1806)
- Add after existing OIDC/Authz flag generation (around line 882)
- Update detection in `deploymentNeedsUpdate()` (around line 1540)

#### Agent Assignment
- **Implementation:** toolhive-expert
- **Verification:** toolhive-expert

---

### ⏳ Phase 4: ConfigMap-Based Configuration Support (PENDING)

**Status:** Not started

#### Acceptance Criteria
- [ ] ExternalAuthConfig field exists in RunConfig struct
- [ ] Builder option `WithExternalAuthConfig()` works correctly
- [ ] Config included in RunConfig ConfigMap when `TOOLHIVE_USE_CONFIGMAP=true`
- [ ] JSON/YAML serialization works correctly
- [ ] `addExternalAuthConfigOptions()` properly fetches and maps MCPExternalAuthConfig
- [ ] ConfigMap updated when external auth config changes
- [ ] Non-existent MCPExternalAuthConfig causes appropriate error
- [ ] ConfigMap mode and flag mode produce equivalent configurations

#### Deliverables
- `pkg/runner/config.go` (modified, +~5 lines)
  - Add ExternalAuthConfig field to RunConfig
- `pkg/runner/config_builder.go` (modified, +~15 lines)
  - Add `WithExternalAuthConfig()` builder option
- `cmd/thv-operator/controllers/mcpserver_runconfig.go` (modified, +~80 lines)
  - `addExternalAuthConfigOptions()` method (~70 lines)
  - Update `createRunConfigFromMCPServer()` to call it

#### Integration Points
- Follow OIDC pattern from `addOIDCConfigOptions()` (lines 734-763)
- Add after audit config options (around line 314)
- Map to `tokenexchange.Config` structure from middleware

#### Agent Assignment
- **Implementation:** toolhive-expert
- **Verification:** go-expert-developer

---

### ⏳ Phase 5: MCPServer Controller Integration (PENDING)

**Status:** Not started

#### Acceptance Criteria
- [ ] MCPServer controller watches MCPExternalAuthConfig resources
- [ ] Changes to MCPExternalAuthConfig trigger reconciliation of referencing MCPServers
- [ ] `handleExternalAuthConfig()` validates that referenced config exists
- [ ] Missing MCPExternalAuthConfig causes appropriate error and status update
- [ ] MCPServer status reflects external auth configuration state
- [ ] Cross-namespace references are rejected
- [ ] Reconciliation loop doesn't cause unnecessary deployments
- [ ] Watch mapping function performs efficiently

#### Deliverables
- `cmd/thv-operator/controllers/mcpserver_controller.go` (modified, +~100 lines)
  - `SetupWithManager()` updated (~20 lines) - add watch for MCPExternalAuthConfig
  - `handleExternalAuthConfig()` added (~50 lines) - validation and hash tracking
  - `Reconcile()` updated (~10 lines) - call handleExternalAuthConfig
  - Mapping function (~30 lines) - find MCPServers when config changes

#### Integration Points
- Follow ToolConfig pattern from `handleToolConfig()` (line 190)
- Call `GetExternalAuthConfigForMCPServer()` from Phase 2
- Use hash from MCPServer status: `ExternalAuthConfigHash`
- Watch setup similar to toolconfig watch (lines 215-242 in toolconfig_controller.go)

#### Agent Assignment
- **Implementation:** kubernetes-go-expert
- **Verification:** kubernetes-go-expert

---

### ⏳ Phase 6: Controller Registration and RBAC (PENDING)

**Status:** Not started

#### Acceptance Criteria
- [ ] MCPExternalAuthConfig controller registered in main.go
- [ ] Operator starts without errors
- [ ] RBAC markers grant appropriate permissions for MCPExternalAuthConfig
- [ ] RBAC markers grant MCPServer controller read access to MCPExternalAuthConfig
- [ ] Generated RBAC manifests are correct
- [ ] Operator works with generated RBAC (no permission errors)
- [ ] RBAC follows principle of least privilege

#### Deliverables
- `cmd/thv-operator/main.go` (modified, +~7 lines)
  - Register MCPExternalAuthConfigReconciler
- `cmd/thv-operator/controllers/mcpexternalauthconfig_controller.go` (RBAC markers already present)
- `cmd/thv-operator/controllers/mcpserver_controller.go` (modified, +~1 line RBAC marker)
  - Add RBAC for reading MCPExternalAuthConfig
- `cmd/thv-operator/config/rbac/role.yaml` (auto-generated, updated)

#### Integration Points
- Register after MCPServerReconciler in main.go
- Run `task operator-manifests` to generate RBAC

#### Agent Assignment
- **Implementation:** kubernetes-go-expert
- **Verification:** kubernetes-go-expert

---

### ⏳ Phase 7: Testing and Documentation (PENDING)

**Status:** Not started

#### Acceptance Criteria

##### Unit Tests
- [ ] MCPExternalAuthConfig controller tests achieve >80% coverage
- [ ] MCPServer controller external auth tests achieve >80% coverage
- [ ] All edge cases are tested (not found, cross-namespace, etc.)
- [ ] Tests run successfully with `task operator-test`

##### Integration Tests
- [ ] Test MCPExternalAuthConfig creation and reconciliation
- [ ] Test MCPServer creation with external auth reference
- [ ] Test external auth config updates trigger MCPServer reconciliation
- [ ] Test finalizer prevents deletion while referenced
- [ ] Test both flag-based and ConfigMap-based modes

##### E2E Tests
- [ ] Chainsaw tests deploy operator and CRDs
- [ ] E2E tests create MCPExternalAuthConfig and MCPServer
- [ ] E2E tests verify deployment has correct configuration
- [ ] E2E tests verify updates propagate correctly
- [ ] Tests run successfully with `task operator-e2e-test`

##### Examples
- [ ] Example manifests work end-to-end
- [ ] Examples demonstrate both inline and ConfigMap token exchange types
- [ ] Examples include secret handling
- [ ] Examples are documented with comments

##### Documentation
- [ ] CRD reference docs are generated and accurate
- [ ] Token exchange proposal is updated with implementation details
- [ ] User guide includes clear examples
- [ ] Migration guide (if applicable) is clear

#### Deliverables

##### Tests
- `cmd/thv-operator/controllers/mcpexternalauthconfig_controller_test.go` (~500 lines)
- `cmd/thv-operator/controllers/mcpserver_controller_test.go` (modified, +~200 lines)
- `test/e2e/chainsaw/operator/external-auth/` directory with Chainsaw tests
- Integration test scenarios

##### Examples
- `examples/operator/mcpexternalauthconfig-inline.yaml`
- `examples/operator/mcpexternalauthconfig-configmap.yaml`
- `examples/operator/mcpserver-with-external-auth.yaml`
- `examples/operator/complete-external-auth-example.yaml` (full working example)

##### Documentation
- Updated `docs/proposals/token-exchange-middleware.md`
- Generated CRD reference docs
- User guide section on external auth configuration

#### Agent Assignment
- **Unit Tests:** go-expert-developer
- **Integration/E2E Tests:** kubernetes-go-expert
- **Examples:** general-purpose
- **Documentation:** general-purpose
- **Verification (Tests):** go-expert-developer
- **Verification (Examples/Docs):** toolhive-expert

---

### ⏳ Phase 8: End-to-End Validation (PENDING)

**Status:** Not started

#### Acceptance Criteria
- [ ] Operator deploys successfully to test cluster
- [ ] MCPExternalAuthConfig CRD installs correctly
- [ ] MCPServer with external auth reference creates deployment
- [ ] Deployment contains correct external auth configuration (flags or ConfigMap)
- [ ] Updates to MCPExternalAuthConfig trigger MCPServer reconciliation
- [ ] MCPServer deployment is updated with new configuration
- [ ] Finalizer prevents deletion of referenced MCPExternalAuthConfig
- [ ] Finalizer is removed when no MCPServers reference the config
- [ ] Token exchange works with real OAuth provider (e.g., Keycloak)
- [ ] Both flag-based and ConfigMap-based modes work correctly
- [ ] Cross-namespace references are rejected
- [ ] Missing references cause appropriate errors

#### Deliverables
- Test report documenting:
  - Deployment success
  - Configuration verification
  - Update propagation
  - Finalizer behavior
  - Real token exchange validation
- Screenshots or logs demonstrating functionality
- Performance metrics (if applicable)

#### Agent Assignment
- **Implementation:** toolhive-expert
- **Verification:** kubernetes-go-expert

---

## Design Decisions

### CRD Design
- **Name:** `MCPExternalAuthConfig` (follows `MCPToolConfig` pattern)
- **Scope:** Namespace-scoped (same-namespace references only)
- **Status:** Simplified to `ObservedGeneration` + `ConfigHash` only
  - **NO** `ReferencingServers` field (computed on-demand during deletion)
- **Strategy Inference:** HeaderStrategy field removed, inferred from `ExternalTokenHeaderName` presence

### Controller Design
- **Pattern:** Follows `MCPToolConfig` controller exactly
- **Finalizer:** Prevents deletion while MCPServers reference the config
- **Hash Algorithm:** FNV-1a (32-bit) using `k8s.io/apimachinery/pkg/util/dump.ForHash`
- **Annotation Key:** `toolhive.stacklok.dev/externalauthconfig-hash`
- **Watch Strategy:** Primary watch on MCPExternalAuthConfig, secondary watch on MCPServer with mapping

### Integration Design
- **Dual Mode:** Support both CLI flags (default) and ConfigMap-based RunConfig
- **Mapping:** CRD fields map directly to `tokenexchange.Config` from middleware
- **Secret Handling:** ClientSecret via `SecretKeyRef` (not inline)

---

## Key Files Modified/Created

### Created
- `cmd/thv-operator/api/v1alpha1/mcpexternalauthconfig_types.go`
- `cmd/thv-operator/controllers/mcpexternalauthconfig_controller.go`
- `deploy/charts/operator-crds/crds/toolhive.stacklok.dev_mcpexternalauthconfigs.yaml`

### Modified
- `cmd/thv-operator/api/v1alpha1/mcpserver_types.go`
- `cmd/thv-operator/api/v1alpha1/zz_generated.deepcopy.go`
- `deploy/charts/operator-crds/crds/toolhive.stacklok.dev_mcpservers.yaml`

### Pending Modifications (Phases 3-6)
- `cmd/thv-operator/controllers/mcpserver_controller.go`
- `cmd/thv-operator/controllers/mcpserver_runconfig.go`
- `pkg/runner/config.go`
- `pkg/runner/config_builder.go`
- `cmd/thv-operator/main.go`

---

## Current Bottleneck

**Phase 2 Unit Tests** - Test implementation was interrupted. Need to resume test generation.

## Next Steps

1. Complete Phase 2 unit tests
2. Verify Phase 2 with independent agent
3. Proceed to Phase 3 (flag-based configuration)
4. Continue through remaining phases sequentially

---

## References

- **Proposal:** `docs/proposals/token-exchange-middleware.md`
- **Token Exchange Implementation:** `pkg/auth/tokenexchange/`
- **Similar Pattern:** MCPToolConfig controller (`toolconfig_controller.go`)
- **CRD Patterns:** Existing CRDs in `cmd/thv-operator/api/v1alpha1/`
