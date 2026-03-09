# rpcli — Runpod CLI for Agents

Agent-first CLI for managing Runpod GPU/CPU infrastructure. All output is structured JSON.

```bash
# Install
go install github.com/agntswrm/rpcli/cmd/rpcli@latest

# Authentication (priority: --api-key flag > RUNPOD_API_KEY env > config file)
export RUNPOD_API_KEY=rpa_...                              # env var (recommended)
rpcli config set-key rpa_...                                # persist to ~/.config/rpcli/config.json
rpcli --api-key rpa_... pod list                            # per-command override

# Global flags (work with any command)
rpcli <command> -o table                                    # output format: json (default), table, yaml
rpcli <command> --dry-run                                   # preview mutations without executing
rpcli <command> --yes                                       # skip confirmation on destructive ops
rpcli <command> --api-key rpa_...                           # override API key for this invocation

# Resources — browse GPUs and CPUs
rpcli resource gpu                                          # in-stock secure cloud GPUs (id, vram, stock status)
rpcli resource cpu                                          # list CPU flavors for pod creation (cpu3c, cpu5m, etc.)
rpcli resource availability                                 # in-stock secure cloud GPUs with pricing + stock
rpcli resource availability "NVIDIA GeForce RTX 4090"       # filter pricing for one GPU type
# pricing fields: securePrice, secureSpotPrice ($/hr per GPU), stockStatus: High/Medium/Low

# Pods — on-demand GPU/CPU instances
rpcli pod create "NVIDIA GeForce RTX 4090" --name my-pod --image runpod/pytorch:2.1.0  # minimal GPU pod
rpcli pod create "NVIDIA A100 80GB PCIe" --name train --image my/img:v1 \
  --gpu-count 4 --container-disk 50 --volume-size 200 \
  --ports "8888/http,22/tcp" --volume-path /workspace \
  --env HF_TOKEN=hf_... --env WANDB_KEY=abc                # multi-GPU with volumes, ports, env vars
rpcli pod create "NVIDIA GeForce RTX 4090" --name spot --image my/img:v1 --spot --bid-price 0.30  # spot/interruptible (cheaper, bid $/hr/GPU)
rpcli pod create "NVIDIA GeForce RTX 4090" --name secure --image my/img:v1 --cloud-type SECURE  # SECURE or COMMUNITY (default ALL)
rpcli pod create "NVIDIA GeForce RTX 4090" --template-id tmpl_abc123  # use existing template instead of --image
rpcli pod create "NVIDIA GeForce RTX 4090" --name my-pod --image my/img:v1 \
  --network-volume vol_abc123                               # attach network volume
rpcli pod create cpu3c-2-4 --name cpu-pod --image ubuntu:22.04  # CPU pod: <flavor>-<vcpus>-<memGB> (use 'resource cpu' for flavors)
rpcli pod list                                              # list all pods
rpcli pod get abc123                                        # get details of one pod
rpcli pod start abc123                                      # start a stopped pod
rpcli pod start abc123 --gpu-count 2                        # start with different GPU count
rpcli pod stop abc123 --yes                                 # stop a running pod
rpcli pod restart abc123 --yes                              # restart a pod
rpcli pod reset abc123 --yes                                # full reset a pod
rpcli pod delete abc123 --yes                               # permanently delete a pod
rpcli pod update abc123 --gpu-count 2                       # change GPU count
rpcli pod update abc123 --volume-size 100                   # resize volume
rpcli pod update abc123 --container-disk 50                 # resize container disk
rpcli pod bid-resume abc123 --bid-price 0.30                # resume spot pod with bid price per GPU
rpcli pod bid-resume abc123 --bid-price 0.50 --gpu-count 2  # bid with different GPU count

# Endpoints — serverless GPU workers
rpcli endpoint create --name my-api --image runpod/pytorch:2.1.0 --gpus ADA_24  # minimal endpoint
rpcli endpoint create --name my-api --image my/img:v1 --gpus ADA_24 \
  --workers-min 1 --workers-max 5 --idle-timeout 10 \
  --container-disk 50 --volume-size 100 --volume-path /models \
  --docker-args "--model-id meta-llama/Llama-3" \
  --env API_KEY=secret --env MODEL_PATH=/models/v1           # full-featured endpoint
rpcli endpoint create --name my-api --template-id tmpl_abc123 --gpus ADA_24  # use existing template
rpcli endpoint create --name my-api --image my/img:v1 --gpus ADA_24 \
  --network-volume vol_abc123                               # attach network volume
rpcli endpoint list                                         # list all endpoints
rpcli endpoint get ep_abc123                                # get endpoint details
rpcli endpoint update ep_abc123 --workers-max 10            # scale workers
rpcli endpoint update ep_abc123 --image new/img:v2 --env NEW_VAR=val  # update image + env
rpcli endpoint update ep_abc123 --network-volume vol_abc123 # attach network volume
rpcli endpoint delete ep_abc123 --yes                       # delete endpoint + auto-cleanup template

# Templates — reusable pod configurations
rpcli template create --name my-tmpl --image my/img:v1 --container-disk 20  # basic template
rpcli template create --name my-tmpl --image my/img:v1 --serverless \
  --env KEY=VALUE --ports "8080/http" --volume-size 50       # serverless template
rpcli template list                                         # list all templates
rpcli template update tmpl_abc123 --image my/img:v2         # update template image
rpcli template update tmpl_abc123 --name new-name --docker-args "--arg"  # update name + docker args
rpcli template delete my-tmpl --yes                         # delete by name (not ID)

# Volumes — persistent network storage
rpcli volume create --name my-vol --size 100 --datacenter US-TX-3  # create 100GB volume
rpcli volume list                                           # list all volumes
rpcli volume update vol_abc123 --name new-name              # rename volume
rpcli volume update vol_abc123 --size 200                   # resize volume
rpcli volume delete vol_abc123 --yes                        # delete volume

# Secrets — securely stored values
rpcli secret create --name MY_TOKEN --value "s3cret"        # create a secret
rpcli secret list                                           # list all secrets (values hidden)
rpcli secret delete sec_abc123 --yes                        # delete a secret

# Registry — private Docker registry credentials
rpcli registry create --name dockerhub --username user --password pass  # add registry creds
rpcli registry list                                         # list registry credentials
rpcli registry update reg_abc123 --username new --password new  # update creds
rpcli registry delete reg_abc123 --yes                      # delete registry creds

# Billing — check spend
rpcli billing pods                                          # pod costs (cost_per_hr, uptime, estimated_cost)
rpcli billing endpoints                                     # endpoint billing (workers min/max)
rpcli billing volumes                                       # volume billing (size, datacenter)

# SSH — keys and pod connections
rpcli ssh list-keys                                         # list SSH keys on your Runpod account
rpcli ssh add-key                                           # auto-generate + register SSH key pair
rpcli ssh add-key --key "ssh-ed25519 AAAA..."               # add specific public key string
rpcli ssh add-key --key-file ~/.ssh/id_ed25519.pub          # add key from file
rpcli ssh info abc123                                       # SSH connection command for a pod (needs 22/tcp port)

# Config & diagnostics
rpcli config show                                           # show current config (API key, paths)
rpcli config set-key rpa_abc123                             # store API key in config file
rpcli doctor                                                # check + auto-fix: API key, SSH keys, account setup
rpcli doctor --dry-run                                      # preview what doctor would fix
rpcli version                                               # print rpcli version

# Gotchas
# - All errors return {"error": {"code": "error_type", "message": "description"}}
# - Destructive ops (stop/delete/restart/reset) require --yes or they will prompt
# - Use --dry-run to preview any mutation before executing
# - GPU names must be full names (e.g. "NVIDIA GeForce RTX 4090"), get valid names from 'rpcli resource gpu'
# - CPU pods use instanceId format: <flavor>-<vcpus>-<memGB> (e.g. "cpu3c-2-4"), get flavors from 'rpcli resource cpu'
# - Multiple GPUs for endpoints: --gpus "NVIDIA GeForce RTX 4090,NVIDIA GeForce RTX 3090"
```
