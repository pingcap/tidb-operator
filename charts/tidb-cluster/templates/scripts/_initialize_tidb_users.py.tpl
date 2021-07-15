import os, sys, MySQLdb
host = '{{ template "cluster.name" . }}-tidb'
permit_host = {{ .Values.tidb.permitHost | default "%" | quote }}
port = 4000
{{- if .Values.tidb.passwordSecretName }}
# We make the assumption if we can connect as the root user w/ the password,
# that we have been run already. If so, exit
with open(os.path.join("/etc/tidb/password", "root"), 'r') as f:
    setup_complete = False
    password = f.read()
    try:
        conn = MySQLdb.connect(host=host, port=port, user='root', password=password, auth_plugin='mysql_native_password', connect_timeout=5)
        setup_complete = True
    except:
        pass
    if setup_complete:
        print("Init job completed successfully on previous run, exiting")
        sys.exit(0)
{{- end }}
conn = MySQLdb.connect(host=host, port=port, user='root', connect_timeout=5)
{{- if .Values.tidb.passwordSecretName }}
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        password = f.read()
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
