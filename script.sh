while [ true ]; do 
    echo "matsko.test.notifications.long.tags 10 `date +%s`" | nc localhost 2003 &
    echo "Sent `date +%s`"
    sleep 10
done
