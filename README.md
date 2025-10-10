# K8s One-click Install

로컬 개발 환경에서 Kubernetes 클러스터(K3s + Colima)와 필수 인프라 도구들을 자동으로 설치하고 설정합니다. ArgoCD를 통한 GitOps 워크플로우를 지원하며, External Secrets Operator로 GitLab과 연동하여 시크릿을 관리합니다.

## 설치 항목

### 자동 설치되는 컴포넌트

1. **[Colima](https://github.com/abiosoft/colima)**: Container runtimes on macOS
   - When using Kubernetes option, [K3s](https://k3s.io/) is installed automatically.
2. **Helm** - Kubernetes 패키지 매니저
3. **MetalLB** - LoadBalancer 타입 서비스에 IP 할당 (Ingress 전 필수)
4. **NGINX Ingress Controller** - 외부 트래픽 라우팅 (MetalLB 의존)
5. **External Secrets Operator (ESO)** - GitLab PAT 기반 시크릿 관리
6. **Cert-Manager** - TLS 인증서 자동 관리
7. **ArgoCD** - GitOps 기반 배포 자동화

### 설치 과정 주요 설정

- 환경 레이블 (dev/staging/prod) 입력 받아 클러스터에 태깅
- GitLab Personal Access Token 입력으로 ESO SecretStore 자동 구성
- Ingress 연결성 검증 후 실패 시 설치 중단 (Critical)

### Colima를 선택한 이유

일반적으로 macOS에서는 K8s 설치를 위해 가상화 환경이 필수적입니다.  
가장 잘 알려진 도구가 [Multipass](https://canonical.com/multipass)고 저도 이것을 사용했었지만, Multipass를 사용할 경우 네트워크 문제가 많이 발생하였습니다.  
가장 문제가 많이 되었던 부분은 [Bridge 네트워크를 설정](https://documentation.ubuntu.com/multipass/latest/how-to-guides/manage-instances/add-a-network-to-an-existing-instance/)하고 NGINX Ingress Controller와 MetalLB을 설치했음에도 간헐적으로 Ingress가 호스트 혹은 외부로 노출이 되지 않던 부분이었으며, Multipass 자체에서 오류가 일어나는 경우도 많았습니다.
https://github.com/canonical/microk8s/issues/908 를 참고해 주세요.

Colima를 사용했을 때는 기본 설정으로도 이런 일이 거의 일어나지 않았기 때문에 macOS에서 더 안정적이라고 판단하였습니다.

## 사용 가능 커맨드

```bash
# 전체 설치
./austinhome install

# 전체 제거 (Colima, Helm, 설정 파일 등 완전 삭제)
./austinhome uninstall
```
