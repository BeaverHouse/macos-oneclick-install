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

### 설치 과정 주요 설정

- 환경 레이블 (dev/staging/prod) 입력 받아 클러스터에 태깅
- GitLab Personal Access Token 입력으로 ESO SecretStore 자동 구성
- Gateway 연결성 검증 후 실패 시 설치 중단 (Critical)

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
# 전체 설치 (K3s + 인프라 + OKE 등록 + kubeconfig export)
austinhome install

# 전체 제거 (Colima, Helm, 설정 파일 등 완전 삭제, 스케줄은 유지)
austinhome uninstall

# 무인 재설치 (uninstall → install → OKE 등록 → kubeconfig export)
austinhome reinstall

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

### 사전 조건

- macOS 자동 로그인 활성화 (System Settings → Users & Groups)
- 최초 1회 `austinhome install` 실행 (OCI 계정 설정, GitLab PAT, OKE cluster 정보가 `~/.austinhome/`에 저장됨)

### 저장되는 설정 (`~/.austinhome/`)

| 파일 | 내용 |
|------|------|
| `gitlab-pat` | GitLab Personal Access Token |
| `oke-cluster-ocid` | OKE 클러스터 OCID |
| `oke-region` | OKE 리전 |

이 설정은 uninstall 시 삭제되지 않으며, `~/.oci/config`도 유지됩니다.

### 로그 확인

```bash
cat /tmp/austinhome-reinstall.log
```

### 갱신이 필요한 경우

- GitLab PAT 만료 → `~/.austinhome/gitlab-pat` 파일 내용 교체
- OCI API key 만료/변경 → `oci setup config` 재실행
