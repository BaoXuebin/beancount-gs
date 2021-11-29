# beancount-gs

![license](https://img.shields.io/github/license/baoxuebin/beancount-gs)

[演示地址](https://beancount.xdbin.com/)
[使用文档](https://www.yuque.com/chuyi-ble7p/beancount-ns)

## 介绍

[beancount](https://github.com/beancount/) 是一个优秀的开源复式记账工具，因为其基于文本记录的特性，难以拓展到移动端；本项目旨在将常见的记账行为提取出接口以供网络调用。

本仓库使用 `Golang` 进行文本的读写和接口服务支持，利用 `bean-query` 获取内容并解析，以 Json 格式返回。并基于已实现的接口内置实现了前端页面（适配移动端）。

![snapshot](https://cdn.xdbin.com/github/beancount-ns/snapshot.png)

## 特性

- [X] 私有部署
- [X] 多用户使用
- [X] 账户管理
- [X] 资产管理
- [X] 统计图表
- [X] 多币种
- [X] 标签
- [X] 投资管理(FIFO)
- [ ] 事件
- [ ] 第三方账单导入

## 如何使用

```shell
go build
```

将打包获得的文件与项目的 `config/`, `public/`, `example/` 三个文件夹放置同一目录，然后执行

# 项目负责人

[@BaoXuebin](https://github.com/BaoXuebin)

# 开源协议

[MIT](https://github.com/BaoXuebin/beancount-gs/blob/main/License) @BaoXuebin