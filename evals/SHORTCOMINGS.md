# rpcli Eval Shortcomings Report

These scenarios scored below 7/10 with SKILL.md context.

## Create spot pod for batch inference (score: 4/10)
**Category:** pod
**Reasoning:** The agent correctly checked spot pricing and attempted spot pod creation with --spot and --bid-price, but no pod was created or verified to exist per ground truth, so the required outcome was not achieved.
- Task not completed: no pod named rpcli-eval-spot was created
- Weak error recovery: did not try changing required parameters (e.g., secure/community selection, region/datacenter, gpuCount/volume) or retry strategy beyond switching GPUs
