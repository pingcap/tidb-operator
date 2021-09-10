import os, MySQLdb
host = '{{ template "cluster.name" . }}-tidb'
permit_host = {{ .Values.tidb.permitHost | default "%" | quote }}
port = 4000
conn = MySQLdb.connect(host=host, port=port, user='root', connect_timeout=5)
{{- if .Values.tidb.passwordSecretName }}
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        lines = f.read().splitlines()
        password = lines[0] if len(lines) > 0 else ""
    if user == 'root':
        conn.cursor().execute("set password for 'root'@'%%' = %s;", (password,))
    else:
        conn.cursor().execute("create user %s@%s identified by %s;", (user, permit_host, password,))
{{- end }}
{{- if or .Values.tidb.initSql .Values.tidb.initSqlConfigMapName }}
with open('/data/init.sql', 'r') as sql:
    for line in sql.readlines():
        conn.cursor().execute(line)
        conn.commit()
{{- end }}
if permit_host != '%':
    conn.cursor().execute("update mysql.user set Host=%s where User='root';", (permit_host,))
conn.cursor().execute("flush privileges;")
conn.commit()
conn.close()
