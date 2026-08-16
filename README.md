# K8s One-click Install

로컬 개발 환경에서 Kubernetes 클러스터(K3s + Colima)와 필수 인프라 도구들을 자동으로 설치하고 설정합니다. ArgoCD를 통한 GitOps 워크플로우를 지원하며, External Secrets Operator로 GitLab과 연동하여 시크릿을 관리합니다.

## 설치 항목

### 자동 설치되는 컴포넌트

1. **[Colima](https://github.com/abiosoft/colima)**: Container runtimes on macOS
   - When using Kubernetes option, [K3s](https://k3s.io/) is installed automatically.
2. **Helm** - Kubernetes 패키지 매니저
3. **MetalLB** - LoadBalancer 타입 서비스에 IP 할당 (Gateway 전 필수)
4. **Gateway API + NGINX Gateway Fabric** - 외부 트래픽 라우팅 (MetalLB 의존, [Ingress NGINX 지원 종료](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/)로 인한 전환)
5. **External Secrets Operator (ESO)** - GitLab PAT 기반 시크릿 관리
6. **Cert-Manager** - TLS 인증서 자동 관리
7. **ArgoCD** - GitOps 기반 배포 자동화

### 설치 과정 주요 사항

- 환경 레이블 (dev/staging/prod) 입력 받아 클러스터에 태깅
- GitLab Personal Access Token 입력으로 ESO SecretStore 자동 구성
- Gateway 연결성 검증 후 실패 시 설치 중단 (Critical)
- OKE 외부 클러스터 등록은 declarative cluster secret 방식 사용 (ArgoCD CLI 미사용). OKE에 `argocd-manager` ServiceAccount + long-lived token Secret을 만들고, K3s ArgoCD에 cluster secret으로 apply.
  - 참고: [ArgoCD Declarative Setup: Clusters](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#clusters), [Bearer token 등록 사례](https://medium.com/pickme-engineering-blog/how-to-connect-an-external-kubernetes-cluster-to-argo-cd-using-bearer-token-authentication-d9ab093f081d)
- Kubeconfig를 export하여 Mac Mini 또는 같은 네트워크 내 기기에서 동일하게 사용할 수 있도록 지원

### Colima + K3s를 선택한 이유

일반적으로 macOS에서는 K8s 설치를 위해 가상화 환경이 필수적입니다.  
가장 잘 알려진 도구가 [Multipass](https://canonical.com/multipass)고 저도 이것을 사용했었지만, Multipass를 사용할 경우 네트워크 문제가 많이 발생하였습니다.  
가장 문제가 많이 되었던 부분은 [Bridge 네트워크를 설정](https://documentation.ubuntu.com/multipass/latest/how-to-guides/manage-instances/add-a-network-to-an-existing-instance/)하고 NGINX Gateway와 MetalLB을 설치했음에도 간헐적으로 Gateway가 호스트 혹은 외부로 노출이 되지 않던 부분이었으며, Multipass 자체에서 오류가 일어나는 경우도 많았습니다.
https://github.com/canonical/microk8s/issues/908 를 참고해 주세요.

Colima를 사용했을 때는 기본 설정으로도 이런 일이 거의 일어나지 않았기 때문에 macOS에서 더 안정적이라고 판단하였습니다.

K3s는 MicroK8s와 함께 간편하게 사용 가능하면서도, Production 환경에서도 사용 가능한 K8s 설치 도구입니다.  
둘 다 사용 경험은 있었고 회사에서는 MicroK8s를 사용했지만, macOS에서는 Multipass 의존성이 강하여 Colima와 쉽게 조합 가능한 K3s를 사용하게 되었습니다.

## 사용 가능 커맨드

```bash
# 업데이트 적용 (명령 없이 실행하면 launch binary와 boot-time reinstall agent 갱신)
austinhome

# 전체 설치 (K3s + 인프라 + OKE 등록 + kubeconfig export)
austinhome install

# 전체 제거 (Colima, Helm, 설정 파일 등 완전 삭제, 스케줄은 유지)
austinhome uninstall

# 무인 재설치 (uninstall → install → OKE 등록 → kubeconfig export)
austinhome reinstall

# kubeconfig export (LAN IP로 변환하여 공유 export 경로에 저장하고 Finder에 표시)
austinhome export-kubeconfig

# 자동 재부팅/재설치 스케줄 등록 (바이너리 설치 포함)
austinhome schedule install

# 스케줄 제거
austinhome schedule remove
```

## 자동 복구

Colima의 메모리 누수 및 네트워크 문제로 인해, 주기적인 재부팅과 재설치를 자동화합니다.

`austinhome schedule` 실행 시:

- **매월 1일 새벽 4시** 자동 재부팅 (launchd daemon)
- **모든 부팅 60초 후** `austinhome reinstall` 자동 실행 (launchd agent)

### 실행 파일 갱신 원칙

빌드된 실행 파일은 `~/Downloads`에 둔 파일을 단일 원본으로 사용합니다.

다만 launchd는 `~/Downloads`의 실행 파일을 직접 실행하지 않습니다. `~/Downloads`에 놓인 파일은 quarantine 또는 TCC 관련 확장 속성을 가질 수 있기 때문에, 스케줄 설정 과정에서 `.local/bin`으로 복사한 launch-safe copy를 실행합니다.

스케줄 설정은 `austinhome schedule install` 또는 실행 파일을 인자 없이 직접 실행해서 갱신합니다. 이 과정은 `~/Downloads`의 원본을 `.local/bin`으로 복사하고, 설치된 복사본이 원본과 byte-for-byte로 같은지 검증합니다.

### 재설치 완료 조건

`austinhome reinstall`은 설치만으로 완료되지 않고, 다음 조건까지 만족해야 성공입니다.

- OKE 클러스터 등록 성공
- OKE cluster secret 적용 후 Argo CD ApplicationSet controller refresh 성공
- 설정된 Argo CD health check application이 `Synced/Healthy`
- kubeconfig export 성공
- export된 kubeconfig가 Finder에 표시됨

### cert-manager GitOps 순서

[cert-manager](https://cert-manager.io)는 현재 [공개 GitOps 저장소](https://github.com/BeaverHouse/cicd)에서 동기화되고 있습니다.
Automated sync와 Self-heal이 활성화되어 있기 때문에 Sync 순서가 맞아야 리소스가 Degraded 상태로 전환되지 않습니다.

예를 들어 DNS credential ExternalSecret은 ClusterIssuer보다 먼저 적용되어야 합니다.

```text
DNS credential ExternalSecret (ex. Route53)    earlier wave
ACME ClusterIssuer                             later wave
```

ACME account Secret은 cert-manager가 런타임에 생성하며 Argo CD가 직접 관리하지 않습니다.

### 사전 조건

- macOS 자동 로그인 활성화 (System Settings → Users & Groups)
- 정전 후 자동 재시작 활성화 (System Settings → Energy)
- 최초 1회 `austinhome install` 실행 (OCI 계정 설정, GitLab PAT, OKE cluster 정보가 `~/.austinhome/`에 저장됨)

그 외에도 [화면 공유](https://support.apple.com/ko-kr/guide/mac-help/mh14066/mac)를 활성화하고, 서버 컴퓨터의 IP를 고정해 두면 유지보수가 편해집니다.

### 저장되는 설정 (`~/.austinhome/`)

| 파일               | 내용                         |
| ------------------ | ---------------------------- |
| `gitlab-pat`       | GitLab Personal Access Token |
| `oke-cluster-ocid` | OKE 클러스터 OCID            |
| `oke-region`       | OKE 리전                     |

이 설정은 uninstall 시 삭제되지 않으며, `~/.oci/config`도 유지됩니다.

### 로그 확인

```bash
cat /tmp/austinhome-reinstall.log
```

### 갱신이 필요한 경우

- GitLab PAT 만료 → `~/.austinhome/gitlab-pat` 파일 내용 교체
- OCI API key 만료/변경 → `oci setup config` 재실행
