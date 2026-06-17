#!/usr/bin/env python3
"""
Figma 设计稿导出工具
团队共享脚本，用于 AI 设计审查

使用方法：
1. 在 Figma 设置 → Personal Access Tokens → 生成 Token
2. 复制 Figma 文件链接（浏览器地址栏）
3. 运行：python3 figma_export.py
4. 把生成的 design_report.md 和 .png 发给 PM

作者：AI 辅助团队
版本：v1.0
"""

import requests
import os
import re
import time

# 配置：建议用环境变量或 .env 文件存储 Token
TOKEN = os.environ.get("FIGMA_TOKEN", "")

class FigmaExporter:
    def __init__(self, token, file_url, output_dir="./figma_export", max_batch=5):
        self.token = token
        self.headers = {"X-Figma-Token": token}
        self.output_dir = output_dir
        self.max_batch = max_batch
        os.makedirs(output_dir, exist_ok=True)
        self.file_key = self._extract_file_key(file_url)
        if not self.file_key:
            raise ValueError("无法解析 Figma 文件 key，请检查链接格式")
        print(f"✅ 文件 key: {self.file_key}")
    
    def _extract_file_key(self, url):
        patterns = [
            r"figma\.com/design/([a-zA-Z0-9]+)",
            r"figma\.com/file/([a-zA-Z0-9]+)",
        ]
        for p in patterns:
            m = re.search(p, url)
            if m:
                return m.group(1)
        return None
    
    def get_file_structure(self):
        url = f"https://api.figma.com/v1/files/{self.file_key}"
        resp = requests.get(url, headers=self.headers, timeout=30)
        if resp.status_code != 200:
            print(f"❌ 获取结构失败: {resp.status_code}")
            print(resp.text)
            return None
        return resp.json()
    
    def extract_pages_and_frames(self, document):
        result = []
        for page in document.get("children", []):
            page_info = {"id": page.get("id"), "name": page.get("name"), "frames": []}
            for child in page.get("children", []):
                if child.get("type") in ["FRAME", "GROUP", "COMPONENT", "INSTANCE"]:
                    bbox = child.get("absoluteBoundingBox", {})
                    page_info["frames"].append({
                        "id": child.get("id"),
                        "name": child.get("name"),
                        "type": child.get("type"),
                        "width": bbox.get("width"),
                        "height": bbox.get("height"),
                    })
            result.append(page_info)
        return result
    
    def export_images(self, node_ids, scale=2, format="png"):
        if not node_ids:
            return {}
        downloaded = {}
        total = len(node_ids)
        for i in range(0, total, self.max_batch):
            batch = node_ids[i:i+self.max_batch]
            print(f"  导出批次 {i//self.max_batch + 1}/{(total-1)//self.max_batch + 1}...")
            node_ids_str = ",".join(batch)
            url = f"https://api.figma.com/v1/images/{self.file_key}"
            params = {"ids": node_ids_str, "scale": scale, "format": format}
            resp = requests.get(url, headers=self.headers, params=params, timeout=60)
            if resp.status_code != 200:
                print(f"  ⚠️ 批次失败: {resp.status_code}")
                time.sleep(3)
                continue
            images = resp.json().get("images", {})
            for node_id, img_url in images.items():
                if not img_url:
                    continue
                safe_id = re.sub(r'[^\w\-]', '_', node_id)
                img_path = os.path.join(self.output_dir, f"{safe_id}.{format}")
                try:
                    img_r = requests.get(img_url, timeout=60)
                    if img_r.status_code == 200:
                        with open(img_path, "wb") as f:
                            f.write(img_r.content)
                        downloaded[node_id] = img_path
                        print(f"    ✅ {os.path.basename(img_path)}")
                    else:
                        print(f"    ⚠️ 下载失败: {img_r.status_code}")
                except Exception as e:
                    print(f"    ⚠️ 异常: {e}")
                time.sleep(0.5)
            if i + self.max_batch < total:
                time.sleep(2)
        return downloaded
    
    def generate_report(self, structure, exported):
        report_path = os.path.join(self.output_dir, "design_report.md")
        lines = []
        lines.append("# Figma 设计稿审查报告")
        lines.append(f"\n> 文件: {self.file_key}")
        lines.append(f"> 导出时间: {time.strftime('%Y-%m-%d %H:%M:%S')}")
        lines.append(f"> 导出图片: {len(exported)} 张")
        lines.append("")
        lines.append("## 页面结构\n")
        for page in structure:
            lines.append(f"### 📄 {page['name']} ({len(page['frames'])} 个画框)")
            lines.append("")
            if page["frames"]:
                lines.append("| 画框名称 | 类型 | 尺寸 | 导出图片 |")
                lines.append("|---------|------|------|---------|")
                for frame in page["frames"]:
                    size = f"{frame['width']}×{frame['height']}" if frame['width'] else "未知"
                    img_name = os.path.basename(exported.get(frame['id'], "未导出"))
                    lines.append(f"| {frame['name']} | {frame['type']} | {size} | {img_name} |")
            else:
                lines.append("*无画框*")
            lines.append("")
        lines.append("## 导出文件清单\n")
        if exported:
            for node_id, path in exported.items():
                lines.append(f"- `{os.path.basename(path)}` → 节点 `{node_id}`")
        else:
            lines.append("*未导出图片*")
        lines.append("")
        lines.append("## 待 AI 审查项\n")
        lines.append("- [ ] 页面布局是否合理")
        lines.append("- [ ] 组件层级是否清晰")
        lines.append("- [ ] 是否有缺失状态（loading / 空态 / 错误态）")
        lines.append("- [ ] 颜色对比度是否符合 WCAG 标准")
        lines.append("- [ ] 字体层级是否一致")
        lines.append("- [ ] 移动端适配是否考虑")
        lines.append("")
        with open(report_path, "w", encoding="utf-8") as f:
            f.write("\n".join(lines))
        print(f"\n📄 报告已生成: {report_path}")
        return report_path
    
    def run(self, page_names=None):
        print("\n🔍 获取文件结构...")
        file_data = self.get_file_structure()
        if not file_data:
            return None
        document = file_data.get("document", {})
        print("📐 分析页面结构...")
        structure = self.extract_pages_and_frames(document)
        if page_names:
            structure = [p for p in structure if p["name"] in page_names]
            print(f"📄 过滤后 {len(structure)} 个页面: {[p['name'] for p in structure]}")
        node_ids = []
        for p in structure:
            for f in p["frames"]:
                node_ids.append(f["id"])
        if len(node_ids) > 20:
            print(f"⚠️ 节点过多 ({len(node_ids)})，只导出前 20 个")
            node_ids = node_ids[:20]
        exported = {}
        if node_ids:
            print(f"\n📸 导出 {len(node_ids)} 个画框...")
            exported = self.export_images(node_ids)
        report = self.generate_report(structure, exported)
        print(f"\n✅ 完成！文件保存在: {self.output_dir}")
        return report, exported

def main():
    print("=" * 60)
    print("  Figma 设计稿导出工具")
    print("  用于 AI 设计审查")
    print("=" * 60)
    
    token = input("\n1. Figma Token (或设置环境变量 FIGMA_TOKEN): ").strip()
    if not token:
        token = TOKEN
        if not token:
            print("❌ 未提供 Token。请设置环境变量 FIGMA_TOKEN 或输入 Token")
            return
    file_url = input("2. Figma 文件链接: ").strip()
    output = input("3. 输出目录 (默认 ./figma_export): ").strip() or "./figma_export"
    page_filter = input("4. 指定页面名（如：首页,大屏，留空则全部）: ").strip()
    pages = [p.strip() for p in page_filter.split(",")] if page_filter else None
    
    try:
        exporter = FigmaExporter(token, file_url, output)
        exporter.run(page_names=pages)
    except Exception as e:
        print(f"\n❌ 错误: {e}")

if __name__ == "__main__":
    main()
