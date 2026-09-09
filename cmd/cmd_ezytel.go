package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hiddify/hiddify-core/v2/ezytel"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	ezytelCacheDir            string
	ezytelJSON                bool
	ezytelDisableInlineImages bool
	ezytelListen              string
)

var commandEzytel = &cobra.Command{
	Use:   "ezytel",
	Short: "Telegram public-channel viewer (EzytelService)",
}

var commandEzytelInfo = &cobra.Command{
	Use:   "info",
	Short: "Fetch channel metadata",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEzytelInfo(cmd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var commandEzytelMessages = &cobra.Command{
	Use:   "messages",
	Short: "Fetch channel message HTML",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEzytelMessages(cmd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var commandEzytelProxyImage = &cobra.Command{
	Use:   "proxy-image",
	Short: "Fetch a proxied Telegram image",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEzytelProxyImage(cmd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var commandEzytelParseChannels = &cobra.Command{
	Use:   "parse-channels",
	Short: "Parse a newline-separated channel list",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEzytelParseChannels(cmd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var commandEzytelServe = &cobra.Command{
	Use:   "serve",
	Short: "Start the Ezytel gRPC server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEzytelServe(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var (
	ezytelChannelID  string
	ezytelBefore     int64
	ezytelLastRead   int64
	ezytelHexURL     string
	ezytelOutput     string
	ezytelHTMLOnly   bool
	ezytelRaw        string
	ezytelRawFile    string
)

func init() {
	commandEzytel.PersistentFlags().StringVar(&ezytelCacheDir, "cache-dir", "", "cache directory (default: tempdir/ezytel-cache)")

	commandEzytelInfo.Flags().StringVar(&ezytelChannelID, "channel", "", "channel id, e.g. durov")
	commandEzytelInfo.Flags().Int64Var(&ezytelLastRead, "last-read", 0, "last read post id for unread badge")
	commandEzytelInfo.Flags().BoolVar(&ezytelDisableInlineImages, "disable-inline-images", false, "return cache paths instead of data: URIs")
	commandEzytelInfo.Flags().BoolVar(&ezytelJSON, "json", false, "output JSON")

	commandEzytelMessages.Flags().StringVar(&ezytelChannelID, "channel", "", "channel id, e.g. durov")
	commandEzytelMessages.Flags().Int64Var(&ezytelBefore, "before", 0, "pagination cursor (0 = latest)")
	commandEzytelMessages.Flags().BoolVar(&ezytelDisableInlineImages, "disable-inline-images", false, "keep proxy.php placeholders in HTML")
	commandEzytelMessages.Flags().BoolVar(&ezytelHTMLOnly, "html-only", false, "print only the html field")
	commandEzytelMessages.Flags().BoolVar(&ezytelJSON, "json", false, "output JSON")

	commandEzytelProxyImage.Flags().StringVar(&ezytelHexURL, "hex", "", "hex-encoded URL without https:// prefix")
	commandEzytelProxyImage.Flags().StringVarP(&ezytelOutput, "output", "o", "", "write image bytes to file")
	commandEzytelProxyImage.Flags().BoolVar(&ezytelJSON, "json", false, "output JSON with base64 data")

	commandEzytelParseChannels.Flags().StringVar(&ezytelRaw, "raw", "", "newline-separated channel list")
	commandEzytelParseChannels.Flags().StringVarP(&ezytelRawFile, "file", "f", "", "read channel list from file")
	commandEzytelParseChannels.Flags().BoolVar(&ezytelJSON, "json", false, "output JSON")

	commandEzytelServe.Flags().StringVar(&ezytelListen, "listen", "127.0.0.1:17079", "gRPC listen address")

	commandEzytel.AddCommand(
		commandEzytelInfo,
		commandEzytelMessages,
		commandEzytelProxyImage,
		commandEzytelParseChannels,
		commandEzytelServe,
	)
	mainCommand.AddCommand(commandEzytel)
}

func newEzytelService() *ezytel.EzytelService {
	return ezytel.NewEzytelService(ezytelCacheDir)
}

func runEzytelInfo(cmd *cobra.Command) error {
	if ezytelChannelID == "" {
		return fmt.Errorf("--channel is required")
	}
	resp, err := newEzytelService().GetChannelInfo(context.Background(), &ezytel.ChannelInfoRequest{
		ChannelId:           ezytelChannelID,
		LastRead:            ezytelLastRead,
		DisableInlineImages: ezytelDisableInlineImages,
	})
	if err != nil {
		return err
	}
	if ezytelJSON {
		return printEzytelJSON(resp)
	}
	fmt.Printf("name: %s\n", resp.Name)
	fmt.Printf("description: %s\n", resp.Description)
	fmt.Printf("avatar_path: %s\n", resp.AvatarPath)
	fmt.Printf("date: %d\n", resp.Date)
	fmt.Printf("date_str: %s\n", resp.DateStr)
	fmt.Printf("newmsg: %s\n", resp.Newmsg)
	fmt.Printf("last_post_id: %d\n", resp.LastPostId)
	fmt.Printf("ok: %v\n", resp.Ok)
	return nil
}

func runEzytelMessages(cmd *cobra.Command) error {
	if ezytelChannelID == "" {
		return fmt.Errorf("--channel is required")
	}
	resp, err := newEzytelService().GetChannelMessages(context.Background(), &ezytel.ChannelMessagesRequest{
		ChannelId:           ezytelChannelID,
		Before:              ezytelBefore,
		DisableInlineImages: ezytelDisableInlineImages,
	})
	if err != nil {
		return err
	}
	if ezytelHTMLOnly {
		fmt.Print(resp.Html)
		return nil
	}
	if ezytelJSON {
		return printEzytelJSON(resp)
	}
	fmt.Printf("last_post_id: %d\n", resp.LastPostId)
	fmt.Printf("channel_avatar: %s\n", resp.ChannelAvatar)
	fmt.Printf("html_bytes: %d\n", len(resp.Html))
	fmt.Print(resp.Html)
	return nil
}

func runEzytelProxyImage(cmd *cobra.Command) error {
	if ezytelHexURL == "" {
		return fmt.Errorf("--hex is required")
	}
	resp, err := newEzytelService().ProxyImage(context.Background(), &ezytel.ProxyImageRequest{
		HexUrl: ezytelHexURL,
	})
	if err != nil {
		return err
	}
	if ezytelJSON {
		out := map[string]string{
			"content_type": resp.ContentType,
			"cache_name":   resp.CacheName,
			"data":         base64.StdEncoding.EncodeToString(resp.Data),
		}
		return printEzytelJSON(out)
	}
	if ezytelOutput != "" {
		return os.WriteFile(ezytelOutput, resp.Data, 0o644)
	}
	if _, err := os.Stdout.Write(resp.Data); err != nil {
		return err
	}
	return nil
}

func runEzytelParseChannels(cmd *cobra.Command) error {
	raw := ezytelRaw
	if ezytelRawFile != "" {
		data, err := os.ReadFile(ezytelRawFile)
		if err != nil {
			return err
		}
		raw = string(data)
	}
	if raw == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		raw = string(data)
	}
	resp, err := newEzytelService().ParseChannels(context.Background(), &ezytel.ParseChannelsRequest{
		Raw: raw,
	})
	if err != nil {
		return err
	}
	if ezytelJSON {
		return printEzytelJSON(resp)
	}
	for _, id := range resp.ChannelIds {
		fmt.Println(id)
	}
	return nil
}

func runEzytelServe() error {
	lis, err := net.Listen("tcp", ezytelListen)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	ezytel.RegisterEzytelServer(s, newEzytelService())
	fmt.Fprintf(os.Stderr, "ezytel gRPC listening on %s\n", lis.Addr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		s.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func printEzytelJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
