# dtool

词库处理工具，词库编码，词库格式转换，词库校验，出简不出全

## 目标

- [ ] 从简码全码混和码表中提取单字全码
- [ ] 从全码表生成出简不出全码表
- [x] 根据单字全码表对词组进行编码（对多编码的字进行笛卡尔积，需要一种编码规则）
- [x] 根据单字全码表对词库进行错码校验
- [ ] 码表格式转换（纯文本的好做，def 参考 asd2fque1 的 DictTool，其他参考深蓝）
- [ ] 根据全拼词库生成双拼词库
- [ ] 词条过滤器（例：词长>9、码长>=5、词频<10）
